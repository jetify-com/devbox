# Devbox performance improvements

Low-hanging performance opportunities, found by timing the up-to-date case and
reading the shellenv, install, and startup code paths.

Effort is approximate **lines of code changed** (production code + tests),
not time. Estimates assume the existing helpers (`filecache`, `xdg`,
`overwriteFileIfChanged`, `errgroup`) are reused.

## Baseline (project already up to date)

| Command | Time |
|---|---|
| raw binary `devbox version` | 10ms |
| raw binary `devbox shellenv` / `global shellenv` | ~90–100ms, of which `nix --version` is ~90ms |
| through the installed launcher `/usr/local/bin/devbox` | +100–200ms |

This matters because `devbox global shellenv` runs on every new terminal and
direnv calls devbox on every `cd`.

## Summary

| # | Item | Win | LOC |
|---|---|---|---|
| 1 | Cache nix version info on disk | ~90ms every command | ~80–120 |
| 2 | Telemetry re-opens the project | 1 extra `devbox.Open` per command | ~30–50 |
| 3 | Cache macOS machine ID | ~20ms every command (macOS) | ~30–50 |
| 4 | Throttle `upload-telemetry` child process | 1 fork + HTTPS per command | ~40–60 |
| 5 | Lightweight `generate direnv --print-envrc` | every direnv reload | ~15–30 |
| 6 | Make devbox its own launcher (instead of a faster launcher script) | ~70ms every command | ~1,500 incl. tests and installer |
| 7 | Early version switch for `devbox_version: "auto"` | one full startup | ~100–150 |
| 8 | Split state hash: nix vs non-nix inputs | ~1s+ per config edit | ~150–250 |
| 9 | Persist narinfo lookups on disk | network round-trips per recompute | ~60–100 |
| 10 | Read profile `manifest.json` instead of `nix profile list` | ~50–100ms per recompute | ~40–70 |
| 11 | Remote plugins: timeout + offline fallback + pinned cache | daily network block | ~60–100 |
| 12 | Parallel package resolution in `devbox update` | N× serial network | ~30–50 |
| 13 | Parallel resolution of lockfile-missing packages | N× serial network | ~60–100 |
| 14 | Don't `--refresh` unchanged packages on update | rebuild of unchanged pkgs | ~20–40 |
| 15 | Batch flake `nix path-info` calls | N nix processes → 1 | ~60–100 |
| 16 | Avoid repeated full recomputes in `Add`/`Update` | 1+ extra recompute | ~30–60 |
| 17 | Small cleanups (4 items) | misc | ~5–20 each |

Items 1–5 total roughly **200–330 LOC** and should make terminal start and
direnv `cd` about 5–10× faster when nothing has changed.

---

## Tier 1: fixed cost on every command

### 1. Cache the nix version info on disk (done in #3020)

**Win:** ~90ms on every command. Could take `shellenv` from ~100ms to ~10–20ms.
**Effort:** ~80–120 LOC.

`EnsureNixInstalled` (`internal/nix/install.go`) and
`Package.IsEnabledOnPlatform` (`internal/devconfig/configfile/packages.go:312`)
both end up calling `(*Nix).Info()` (`nix/nix.go:164`), which runs
`nix --version --debug`. The result is cached only per process via
`infoOnce`.

Fix:
- Persist the parsed `Info` under the XDG cache dir (e.g. via `filecache`).
  Key it on the resolved nix binary path (an immutable `/nix/store/...` path)
  plus the mtimes of the `nix.conf` files that `--debug` reports. ~60–90 LOC
  including tests.
- In `IsEnabledOnPlatform`, return `true` early when neither `Platforms` nor
  `ExcludedPlatforms` is set, so `nix.System()` is never called in the common
  case. ~3–5 LOC.

### 2. Telemetry re-opens the whole project after every command (done in #3021)

**Win:** one extra `devbox.Open` per command (re-parses devbox.json and
devbox.lock, reloads plugins, which can hit the network).
**Effort:** ~30–50 LOC.

`internal/boxcli/midcobra/telemetry.go:57` calls `getPackagesAndCommitHash`,
which calls `devbox.Open`. This runs before checking whether telemetry is
enabled, so it still happens with `DO_NOT_TRACK` set.

Fix:
- Return early from the post-run hook when telemetry is disabled. ~5 LOC.
- Reuse the `Devbox` the command already opened (stash it on the command
  context) instead of opening a second one. ~25–40 LOC.
- Related bug: `c.Flag("config")` (line 85) is looked up on the root command,
  so it is always nil and `--config` is ignored. Look it up on the resolved
  subcommand instead. ~2–5 LOC.

### 3. Cache the macOS machine ID (done in #3021)

**Win:** ~20ms per command on macOS.
**Effort:** ~30–50 LOC.

`internal/telemetry/telemetry.go:64` calls `machineid.ProtectedID`, which runs
`ioreg` every time. Fix: compute it once and store it in the XDG state dir;
read the file on subsequent runs.

### 4. Throttle the `devbox upload-telemetry` child process (done in #3021)

**Win:** removes a second devbox process + HTTPS calls on every command (every
direnv `cd`).
**Effort:** ~40–60 LOC.

`internal/telemetry/telemetry.go:90` starts `devbox upload-telemetry` after
every successful command. Fix: record the last upload time in the state dir and
only spawn the uploader when it was more than N minutes ago, or when the
buffered event count exceeds a threshold.

### 5. Make `generate direnv --print-envrc` cheap (done in #3022)

**Win:** every direnv reload.
**Effort:** ~15–30 LOC.

`devbox generate direnv --print-envrc` only prints a fixed template, but it
inherits `PersistentPreRunE: ensureNixInstalled` from `generateCmd`
(`internal/boxcli/generate.go:55`) and runs the telemetry hooks. Fix: override
`PersistentPreRunE` on the direnv subcommand (or check the flag) to skip the
nix check, and skip telemetry for it.

### 6. Make devbox its own launcher (done in #3024)

**Win:** about 70ms on every command through `/usr/local/bin/devbox`. With
`devbox version`, the bash launcher took 78ms. The native launcher takes 8ms
when the installed binary is the current version, about the same as running
the binary directly, and 17ms when it switches to another version.
**Effort:** ~1,500 LOC including tests and the new installer, all in this repo.

The bash launcher forks about 10 subshells (`tr`, `uname`, `date -r`, `cat`)
and chains three `exec env` calls. When its version cache is more than 24h old
it also does a blocking `curl` with no timeout. Rather than speed up the
script in the launcher repo, release builds of devbox now do the launcher's
job themselves (`internal/launcher`), and version checks run in a detached
child process. See `specs/native-launcher.md` for the design, compatibility
with older versions, and the rollout. The rollout's remaining step is
publishing the new installer.

### 7. Switch versions early for `devbox_version: "auto"` (done in #3023)

**Win:** avoids paying for two full startups on every invocation.
**Effort:** ~100–150 LOC.

The project version check runs inside `devbox.Open`, which happens after the
nix check. Switching versions then re-execs through the bash launcher, and the
decision isn't remembered between invocations. Fix:
- Read `devbox_version` in a root middleware that runs before any `PreRunE`.
- Exec the already-downloaded binary directly instead of going through the
  launcher.
- Cache the resolved version so later invocations don't re-resolve it.

---

## Tier 2: after any `devbox.json` edit

### 8. Split the state hash into nix-relevant and other inputs

**Win:** ~1s or more on every config edit that doesn't touch packages (biggest
win for editing workflows).
**Effort:** ~150–250 LOC.

`Devbox.ConfigHash` (`internal/devbox/devbox.go:199`) hashes the entire config.
Editing a script, env var, or init_hook marks everything stale, which triggers
narinfo HEAD requests, `nix path-info`, `nix print-dev-env`, and a profile
sync.

Fix:
- Compute two hashes: nix inputs (packages, lockfile, plugins, flakes, nixpkgs
  commit, env that affects the build) and everything else.
- Store both in `stateHashFile` (`internal/lock/statehash.go`).
- When only the "everything else" hash changed, regenerate scripts/env and
  reuse the cached `print-dev-env` output.

### 9. Persist narinfo lookups on disk

**Win:** skips HEAD requests to cache.nixos.org on every recompute.
**Effort:** ~60–100 LOC.

`internal/devpkg/narinfo_cache.go:218` caches results in a `sync.Map`, which
only lasts for the process. Fix: persist positive results in a `filecache`
keyed by `uri/hash` with a long TTL (a narinfo for a given store hash is
effectively immutable). Keep misses in memory only or with a short TTL.

### 10. Read the profile manifest instead of running `nix profile list` (done in #3030)

**Win:** ~50–100ms per recompute.
**Effort:** ~40–70 LOC.

`nixprofile.ProfileListItems` (`internal/nix/nixprofile/profile.go:28`) shells
out to `nix profile list --json`. Fix: read
`<profile>/manifest.json` directly and parse it with the existing structs.
Fall back to the command for unknown manifest versions.

### 11. Remote plugins: timeout, offline fallback, better caching (done in #3029)

**Win:** removes a once-a-day blocking network call.
**Effort:** ~60–100 LOC.

- GitHub plugins use a bare `&http.Client{}` with no timeout
  (`internal/plugin/github.go:101`). Add a timeout. ~5 LOC.
- When the 24h cache has expired and the fetch fails, fall back to the stale
  cached content. ~20–40 LOC.
- Git plugins pinned to a commit clone once per file and still expire after
  24h (`internal/plugin/git.go:194`). Cache commit-pinned content
  indefinitely, and clone once per repo/ref rather than once per file.
  ~30–50 LOC.

---

## Tier 3: `install` / `add` / `update`

### 12. Resolve packages in parallel in `devbox update`

**Effort:** ~30–50 LOC.

`updatePendingPackages` (`internal/devbox/update.go:135`) resolves each package
serially. Fix: resolve in an `errgroup` with bounded concurrency, then apply
lockfile writes serially. Preserve the existing error semantics (flakes
warn-and-continue, versioned packages abort).

### 13. Resolve lockfile-missing packages in parallel

**Effort:** ~60–100 LOC.

`FillNarInfoCache` (`internal/devpkg/narinfo_cache.go:56`) calls
`isEligibleForBinaryCache` → `resolve()` serially because `resolve()` is not
concurrency-safe. Fix: make lockfile resolution safe to call concurrently
(lock around lockfile mutation, do the search request outside the lock), then
resolve in parallel. Also speeds up `devbox add a b c`.

### 14. Don't `--refresh` unchanged packages on update

**Effort:** ~20–40 LOC.

`devbox update` rebuilds every package with `--refresh`
(`internal/devbox/update.go:78`), even when its lockfile entry didn't change.
Fix: only pass `--refresh` for packages whose resolved version/hash changed.

### 15. Batch flake `nix path-info` calls

**Effort:** ~60–100 LOC.

`Package.GetStorePaths` (`internal/devpkg/package.go:777`) runs one
`nix.StorePathsFromInstallable` process per installable. Fix: collect
installables across packages and run a single `nix path-info` call, then map
results back. Keep the per-package error handler for the failure case (retry
individually to attribute the error).

### 16. Avoid repeated full recomputes in `Add` / `Update`

**Effort:** ~30–60 LOC.

When `Add` replaces a package with the same canonical name, it calls
`d.Remove` (`internal/devbox/packages.go:117`), which runs its own
`ensureStateIsUpToDate`, and then `Add` recomputes again. `Update` can call
`Add` per package, each with its own recompute. Fix: factor out a
config-only remove/add, and run a single `ensureStateIsUpToDate` at the end.

### 17. Smaller cleanups

Each ~5–20 LOC:
- Remove the `nix.FlakeUpdate` call in `internal/devbox/update.go:100`; its
  own comment says it probably isn't needed.
- Run `nix flake prefetch` once instead of twice
  (`internal/nix/nixpkgs.go:39-78`): use `--json` on the first call and save
  the location from its output.
- `RunXPaths` (`internal/devbox/devbox.go:1196`) deletes and recreates
  `.devbox/virtenv/runx/bin` on every env compute. Skip it when there are no
  runx packages, and otherwise only rewrite changed links.
- `WriteScriptsToFiles` (`internal/shellgen/scripts.go:41`) rewrites all
  scripts on every `devbox run`. Use the existing `overwriteFileIfChanged`
  (`internal/shellgen/generate.go:106`).
- `IsUpToDateAndInstalled` runs 2–3 times per shellenv
  (`internal/devbox/packages.go:310`, `internal/devbox/devbox.go:877`).
  Memoize it per `Devbox` instance.

---

## Not worth chasing

`vercheck` does no I/O on the hot path. `init()` functions and template parsing
are cheap. `nix path-info --offline` and `nix build` are already batched, and
nixpkgs resolution is already cached.

## Dead code found along the way

- `checkOldEnvrc` (`internal/devbox/devbox.go:742`) never runs, so the
  stale-`.envrc` warning has silently stopped working. ~10–20 LOC to rewire or
  delete.
- `doLogCommand` (`internal/boxcli/log.go`) always returns an error, even for
  valid events. Nothing sets the variable that enables it, so it's unused.
  ~10–30 LOC to delete.

## Suggested order

1. Items 1–5 (~200–330 LOC): biggest win for terminal start and direnv `cd`.
2. Item 8 (~150–250 LOC): biggest win for editing `devbox.json`.
3. Item 12 (~30–50 LOC): easiest win for `devbox update`.
4. The rest as time allows. Item 6 is done except for publishing the new
   installer (see `specs/native-launcher.md`).
