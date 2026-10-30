# Devbox as its own launcher

Replace the bash launcher (`releases.jetify.com/devbox`, installed at
`/usr/local/bin/devbox`) with the devbox binary itself. The binary that's
installed in `/usr/local/bin` (the "front binary") picks the devbox version to
run, downloads it if needed, and either runs the command itself or execs that
version. All of this then lives in this repo, in `internal/launcher`.

**Status:** implemented, except publishing the new installer
(`scripts/install.sh`) to `get.jetify.com/devbox` and retiring the launcher in
the axiom repo. See [Rollout](#rollout).

## Why

- The bash launcher adds about 70ms to every command, mostly from subshells
  and chained `exec env`. When its version cache is more than 24h old, it also
  does a blocking `curl` with no timeout.
- The launcher lives in another repo (`axiom/installers/devbox`), so fixing it
  means changing two repos that have to stay in sync. `vercheck` already
  mirrored launcher internals, such as the `current-version` file and
  `INSTALL_DIR`.

Measured with `devbox version` (median of 30 runs, macOS arm64):

| Path | Time |
|---|---|
| binary run directly | 9.0ms |
| bash launcher, then the binary | 78.2ms |
| native launcher, front binary is the current version (no exec) | 8.4ms |
| native launcher, then another version | 17.3ms |

## Terms

- **Front binary:** the devbox binary at `/usr/local/bin/devbox`.
- **Install dir:** `$HOME/.cache/devbox/bin`, where each version lives in
  `<version>_<os>_<arch>/devbox`. It's always under `$HOME`, even when
  `XDG_CACHE_HOME` is set, to match the bash launcher.
- **Launcher version:** `launcher.Version`, separate from the devbox version.
  The front binary passes it to the version it runs as `LAUNCHER_VERSION`. The
  bash launcher's versions are below 1.0.0, and the native launcher starts at
  1.0.0. Bump it only when installed launchers need to change (see
  [Updates](#updates)).
- **Selected version:** the version that should run: `DEVBOX_USE_VERSION` if
  set, otherwise `current-version`.

## How the launcher works

`launcher.Main` runs first in `boxcli.Main`, before cobra, telemetry, or
anything that opens a project.

1. **Only in release builds.** It does nothing unless the
   `build.Launcher=true` ldflag is set, which only `.goreleaser.yaml` sets.
   Dev builds and package-manager builds (nixpkgs, Homebrew) stay plain
   binaries, so the package manager stays in charge of their version.
2. **Not for installed versions.** It does nothing when the binary is in the
   install dir. Launchers (bash or native) only run binaries from there, so
   such a binary is already the version a launcher picked. This rule, rather
   than an env var guard, prevents exec loops, and env vars that leak into a
   `devbox shell` can't confuse it.
3. **Fast path.** If the selected version is the front binary's own version,
   it sets the launcher env vars and runs the command in-process.
4. **Otherwise** it execs the selected version from the install dir. If that
   version isn't installed, it downloads the release archive and its
   `checksums.txt` from GitHub first, checks the archive's SHA-256, unpacks
   it into a temp dir, and renames that into place.
5. **Version checks don't block.** If `current-version` was last modified
   more than 24h ago, it resets the file's modification time and starts a
   detached `devbox __launcher-refresh` process. That process saves the
   latest version to `latest-version`. With `auto_update` on, it also
   downloads that version and then makes it the current version. A goroutine
   wouldn't work, because a short command exits before it finishes.
6. **No current version.** This happens on the first run after an install
   that didn't write `current-version`, or after an older devbox's `devbox
   version update` deleted the file. Like the bash launcher, it waits (up to
   5s) for the latest version and saves it as the current version. If that
   fails, it runs itself without saving anything, so the next run tries again.

### Compatibility with older versions

The front binary runs older versions (anything a project or user pins), and
older binaries may run under the bash launcher, so both sides keep the bash
launcher's contract:

- `DEVBOX_USE_VERSION` picks the version, including for the `"auto"` policy's
  re-exec through `LAUNCHER_PATH` (`vercheck/project.go`).
- `current-version` and `latest-version` in `$XDG_CACHE_HOME/devbox`, and the
  `auto_update` and `channel` files in `$XDG_CONFIG_HOME/devbox`, keep their
  meaning.
- The launcher sets `LAUNCHER_PATH` (its own path), `LAUNCHER_VERSION`, and
  `DEVBOX_LATEST_VERSION`. Older binaries compare `LAUNCHER_VERSION` to their
  expected 0.2.2, so 1.0.0 doesn't make them ask for a new launcher.
- The install dir layout is unchanged, so versions that the bash launcher
  already downloaded are reused, and `vercheck` execs them directly as before
  (#3023). 32-bit ARM uses `armv7l` like `uname -m` and the release archives.

## Updates

Most updates never touch the front binary, so they never need sudo.

- **`devbox version update`** gets the latest version, downloads it into the
  install dir, and saves it as the current and latest version. The front
  binary then execs it. It no longer reruns the installer or the launcher.
- **The front binary is replaced only when the launcher version is bumped.**
  When `LAUNCHER_VERSION` is older than `expectedLauncherVersion` (now
  `launcher.Version`), devbox prints "New launcher available" and `devbox
  version update` also replaces the front binary.
- **`devbox version update --launcher`** replaces the front binary even when
  the launcher is up to date. It's for people who want the fast path back,
  meaning no exec on every command, and don't mind sudo.
- **How it's replaced:** with the latest version's binary, or with the running
  binary if the latest is older (for example on a channel), because the new
  front binary must have the native launcher. It copies to a temp file next to
  the destination and renames it, resolving symlinks first. If the directory
  isn't writable, it runs the same copy and rename with `sudo`.
- **Not installed with the installer:** without `LAUNCHER_PATH`, `devbox
  version update` returns an error that says to update devbox with the
  package manager it came from.

## Migration and installer

- **Existing bash-launcher users** see "New launcher available: v0.2.2 ->
  v1.0.0" once this release is the version they run. `devbox version update`
  then replaces `/usr/local/bin/devbox` (the script) with the binary itself,
  without the installer. So the migration doesn't depend on publishing the new
  installer. Users who never update keep the bash launcher, which still works.
- **The notice is throttled** to once a day using a `launcher-notice` file in
  the cache. It isn't printed (and doesn't use up the day's notice) when
  `--quiet` discards stderr. It's still skipped for `shellenv` and the other
  commands in `commandSkipList`, and it never prompts.
- **The new installer**, `scripts/install.sh`, gets the latest version,
  downloads the archive and `checksums.txt` from GitHub, checks the archive,
  and installs the binary to `/usr/local/bin/devbox`. It uses sudo only when
  that directory isn't writable. It writes `current-version` and
  `latest-version`, so the first run doesn't wait for the network. It keeps
  the old installer's `-f` flag and its curl/wget/fetch support. It always
  installs the latest version, because the front binary honors
  `DEVBOX_USE_VERSION` (which the generated Dockerfiles set) by downloading
  that version on the first run.

## Rollout

1. Ship this in a release. New and existing installs keep working as before
   until users run `devbox version update`.
2. After that release is the stable version, publish `scripts/install.sh` to
   `get.jetify.com/devbox` (it's currently published from
   `axiom/installers/devbox/install.sh`). It needs a release with the native
   launcher, because it installs the latest release as the front binary.
   Older devbox versions' `devbox version update` runs this installer too,
   through `get.jetpack.io/devbox`, which redirects to the same script.
3. Keep `releases.jetify.com/devbox` (the bash launcher) published for
   anything that downloads it directly. Later, retire it and
   `axiom/installers/devbox`.
