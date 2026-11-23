# Package definitions: installing packages directly from the vendor

Status: in progress. This document is both the format specification and the
living implementation plan. Update the status table at the bottom as work
lands.

## Goal

Let a Devbox project install a tool from the upstream vendor's own binary
distribution (go.dev, nodejs.org, GitHub releases, ...) instead of nixpkgs,
driven by a small declarative *package definition* file. Go is the first
package (see [`pkgs/go/go.pkg.jsonc`](../pkgs/go/go.pkg.jsonc)). Later, the
search service returns these definitions so that `devbox add go@1.22`
transparently picks the vendor build.

Prior art this design borrows from:

- Devbox `runx:` packages already fetch GitHub release archives into a
  user-level cache, skip the generated flake, and get symlinked onto `PATH`.
  Package definitions generalize that path.
- Devbox plugins (`plugin.json`) provide `env`, `shell.init_hook`,
  `create_files` and Go-template variables such as `{{ .Virtenv }}`.
  Definitions reuse this for activation.
- proto's TOML plugins and the aqua registry use per-platform URL templates,
  version sources and checksum sources. The aqua registry is also a seed
  corpus for bulk authoring.

## Referencing a definition

A definition is a JSON file named `<name>.pkg.json` or `<name>.pkg.jsonc`.
Both extensions are parsed as hujson (comments and trailing commas allowed);
prefer `.jsonc` when the file has comments so editors highlight it correctly.
A project references it with the `pkg:` scheme:

```sh
devbox add pkg:./go.pkg.jsonc@1.22
devbox add pkg:https://raw.githubusercontent.com/jetify-com/devbox/main/pkgs/go/go.pkg.jsonc@1.22
```

The string is `pkg:<ref>@<version>`. The version is split at the last `@`,
exactly like `name@version` today, so the existing config and lockfile code
paths keep working. `<ref>` is one of:

| Form | Example | Notes |
|---|---|---|
| relative path | `./go.pkg.jsonc`, `../defs/go.pkg.jsonc` | Relative to the directory of `devbox.json`. |
| absolute path | `/opt/defs/go.pkg.jsonc` | |
| `https://` URL | `https://example.com/go.pkg.jsonc` | Content is cached by hash; the lockfile pins the hash. |
| `file://` URL | `file:///tmp/go.pkg.jsonc` | Mainly for tests. |
| `github:` | `github:jetify-com/devbox/pkgs/go/go.pkg.jsonc` | Planned. Sugar for the raw.githubusercontent.com URL. |

If no `@version` is given, `devbox add` appends `@latest`.

Naming alternatives that were considered: `vendor:` with `go.vendor.json`
reads well but collides with Go's `vendor/` convention; `.ext` is too generic
and has no editor support; `devpkg:` is unambiguous but verbose. JSON was
chosen over TypeScript because it needs no runtime, hashes cleanly into the
lockfile, and uses the same parser as `devbox.json` and `plugin.json`.
Dynamic behaviour goes through Go templates and an explicit `script` escape
hatch.

## File format

```jsonc
{
  "schema": 1,
  "name": "go",
  "description": "The Go programming language",
  "homepage": "https://go.dev",
  "license": "BSD-3-Clause",

  // Nix system names, same vocabulary as the devbox.json "platforms" field.
  "platforms": ["aarch64-darwin", "x86_64-darwin", "aarch64-linux", "x86_64-linux"],

  // Map nix OS/arch to the vendor's spelling. These are the defaults.
  "os_map":   { "darwin": "darwin", "linux": "linux" },
  "arch_map": { "x86_64": "amd64", "aarch64": "arm64" },

  // How to list versions. Used to resolve "latest", a prefix ("1.22") or an
  // exact version to one concrete version at `devbox add`/`devbox update`.
  "versions": {
    "type": "http_json",
    "url": "https://go.dev/dl/?mode=json&include=all",
    "select": ".[] | select(.stable) | .version | ltrimstr(\"go\")"
  },

  // How to install one version on one system.
  "install": {
    "type": "archive",
    "url": "https://go.dev/dl/go{{.Version}}.{{.OS}}-{{.Arch}}.tar.gz",
    "checksum": {
      "type": "http_json",
      "url": "https://go.dev/dl/?mode=json&include=all",
      "select": ".[].files[] | select(.filename == \"{{.Filename}}\") | .sha256"
    },
    "strip_components": 1,
    "overrides": {}
  },

  // Directories (relative to the install root) to put on PATH, and the
  // executables expected after install. "binaries" is used by validation
  // tooling and post-install smoke tests.
  "bin": ["bin"],
  "binaries": ["go", "gofmt"],

  // Activation. Same keys and template variables as plugin.json, plus
  // {{ .InstallDir }} and {{ .Version }}.
  "env": {},
  "shell": { "init_hook": [] },
  "create_files": {},

  // Shell commands run once after a version is installed, with the install
  // root as the working directory.
  "hooks": { "post_install": [] }
}
```

### Top-level fields

| Field | Required | Description |
|---|---|---|
| `schema` | yes | Format version. Currently `1`. Devbox rejects unknown values. |
| `name` | yes | Package name. `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`. Used for the cache directory and the plugin virtenv directory. |
| `description`, `homepage`, `license` | no | Informational. `devbox info` showing `homepage` is planned. |
| `platforms` | yes | Non-empty list of nix system names the definition supports. |
| `os_map`, `arch_map` | no | Overrides for the `{{.OS}}` and `{{.Arch}}` template variables. Keys are the nix OS (`darwin`, `linux`) and nix arch (`x86_64`, `aarch64`). |
| `versions` | yes | A version source. See below. |
| `install` | yes | An install strategy. See below. |
| `bin` | no | Directories relative to the install root that go on `PATH`. Default `["bin"]`. |
| `binaries` | no | Executable names expected on `PATH` after install. |
| `env`, `shell`, `create_files` | no | Plugin-compatible activation. See Activation. |
| `hooks.post_install` | no | Shell commands run once per installed version. |

### Version sources (`versions`)

| `type` | Fields | Behaviour |
|---|---|---|
| `static` | `versions: [string]` | The list is the universe of versions. |
| `http_json` | `url`, `select` | GET `url`, parse JSON, evaluate the jq expression in `select`. Every string (or number) it emits is a version. |
| `github_releases` | `repo` (`owner/name`), `strip_prefix` (default `"v"`) | List releases through the GitHub API, skip drafts and pre-releases, strip the prefix from tag names. Honors `GITHUB_TOKEN`. |
| `script` | `run` | Planned. Run `run` with `sh -c`; one version per output line. |

Version strings are compared as dotted numeric segments, falling back to
string comparison. Versions containing a hyphen (`1.23.0-rc1`) are treated as
pre-releases and only match when requested exactly.

Matching a requested version against the list:

| Requested | Match |
|---|---|
| `latest` | The highest non-pre-release version. |
| `1.22` | The highest version equal to `1.22` or starting with `1.22.`. |
| `1.22.3` | That version exactly (also by the prefix rule, so `1.22.3` matches `1.22.3.1`-style versions if the exact one is absent). |

### Install strategies (`install`)

`install.type` picks how a version lands on disk. `install.overrides` is a
map from nix system to a partial `install` object whose fields replace the
defaults for that system (for example a different `url` or `strip_components`
on one platform).

| `type` | Fields | Behaviour |
|---|---|---|
| `archive` | `url`, `checksum`, `strip_components` (default `0`) | Download the archive, verify it, extract it into the install root. Supports `.tar.gz`, `.tgz`, `.tar.xz`, `.tar.bz2`, `.tar.zst` and `.zip`. |
| `binary` | `url`, `checksum`, `bin_name` (default: `name`) | Download a single executable to `<root>/bin/<bin_name>` and mark it executable. |
| `manager` | `requires`, `env`, `install`, `locate` | Planned. Delegate to another definition that owns version management (see below). |
| `script` | `run` | Planned. Escape hatch: run `run` with the install root as `$DEVBOX_PKG_INSTALL_DIR`. |

Why two real strategies: `archive` covers more than it first appears. Go,
nodejs, deno, bun, zig, Temurin JDK and python (via python-build-standalone,
the same builds uv installs) all ship per-platform archives, so fnm, nvm and
pyenv are not needed for them. `manager` is for ecosystems where the vendor
ships no usable archive or the manager is the ecosystem's contract: rustup
for rust, ruby (no official binaries), php on Linux.

A manager-backed definition will look like this once implemented:

```jsonc
{
  "name": "rust",
  "versions": { "type": "github_releases", "repo": "rust-lang/rust", "strip_prefix": "" },
  "install": {
    "type": "manager",
    "requires": "pkg:https://.../rustup.pkg.json@latest",
    "env": { "RUSTUP_HOME": "{{ .CacheDir }}/rustup", "CARGO_HOME": "{{ .CacheDir }}/cargo" },
    "install": "rustup toolchain install {{.Version}} --profile minimal",
    "locate": "rustup run {{.Version}} rustc --print sysroot"
  },
  "bin": ["bin"],
  "env": { "RUSTUP_TOOLCHAIN": "{{.Version}}" }
}
```

- `requires` is a single, non-transitive dependency on another definition,
  installed first. Keeping it to one level avoids building a resolver.
- The manager's home directories are pointed into the Devbox cache through
  `install.env`, so managed versions live next to archive installs and future
  cache GC sees them.
- Checksum verification is delegated, so the lockfile records
  `verified_by: "manager"` instead of per-system hashes, and `locate` gives
  the install root at install time.

Go's native switching (`GOTOOLCHAIN`, Go 1.21+) is expressible as a `manager`
definition whose manager is the archive-installed bootstrap `go`. It is not
the default for Go: downloads are once per version either way, and the
toolchain route needs a bootstrap, leaves `gofmt` on the bootstrap version,
and overrides the `toolchain` line in go.mod. Its one advantage is
verification through sum.golang.org, so it stays an alternate definition.

### Checksums (`install.checksum`)

| `type` | Fields | Behaviour |
|---|---|---|
| `http_json` | `url`, `select` | GET `url`, evaluate the jq expression; the first string result is the hex sha256. |
| `sha256_url` | `url` | GET `url`. The body is either a bare hex digest or a `sha256sum`-style list (`<hex>  <filename>` per line) matched by `{{.Filename}}`. Covers nodejs `SHASUMS256.txt` and the common `<asset>.sha256` convention. |
| `none` | | No vendor-published checksum. `devbox add` records the digest it observes for the current system (trust on first use). Other systems are filled in lazily when a machine of that system installs, matching how nix store paths are backfilled today. |

Checksums are mandatory when installing from an existing lockfile entry that
records a digest: the downloaded artifact must match the digest recorded for
the system, or the install fails. A system entry without a digest (checksum
`none`) installs with a warning and records the observed digest (trust on
first use).

### Template variables

Templates use Go `text/template` with `missingkey=error`. Available
variables:

| Variable | Where | Value |
|---|---|---|
| `{{.Version}}` | everywhere | The resolved version, e.g. `1.22.3`. |
| `{{.System}}` | everywhere | Nix system, e.g. `aarch64-darwin`. |
| `{{.OS}}`, `{{.Arch}}` | everywhere | Vendor spelling after `os_map`/`arch_map`. |
| `{{.URL}}`, `{{.Filename}}` | `checksum`, `hooks` | The rendered artifact URL and its last path segment. |
| `{{.InstallDir}}` | `env`, `shell`, `create_files`, `hooks` | Install root for this version and system. |
| `{{.CacheDir}}` | `env`, `shell`, `hooks`, `install.env` | `$XDG_CACHE_HOME/devbox/pkgs`. |

Functions: `trimPrefix`, `trimSuffix`, `replace`, `lower`, `upper`.

### Activation

`env`, `shell.init_hook` and `create_files` have the same schema as in
`plugin.json`. Devbox turns each package definition into a plugin whose source
is the definition, so init hooks and created files work through the existing
plugin manager with no new machinery. `create_files` sources are resolved
relative to the definition file; they are only supported for local
definitions.

No deactivate hook exists on purpose: Devbox never mutates the parent shell.
Every `devbox shell` and `devbox run` recomputes `PATH` and env from the
lockfile, so changing the version in `devbox.json` is the switch. This holds
for every install strategy.

## Lockfile

```jsonc
"pkg:./go.pkg.jsonc@1.22": {
  "source": "pkg-def",
  "version": "1.22.3",
  "resolved": "pkg:./go.pkg.jsonc@1.22.3",
  "definition_hash": "9f2c...",  // lowercase hex sha256, no prefix
  "systems": {
    "aarch64-darwin": { "url": "https://go.dev/dl/go1.22.3.darwin-arm64.tar.gz", "sha256": "..." },
    "x86_64-linux":   { "url": "https://go.dev/dl/go1.22.3.linux-amd64.tar.gz",  "sha256": "..." }
  }
}
```

- `definition_hash` is the sha256 of the definition file content. For remote
  definitions it pins the content: Devbox caches the bytes by hash and
  refuses to install from a definition whose hash differs. For local
  definitions it is informational; edits are picked up through the state
  hash and trigger a reinstall.
- `systems.*.url` and `systems.*.sha256` are new `omitempty` fields on the
  existing per-system struct. `lockfile_version` stays `1`.
- Older Devbox binaries reject the `pkg:` package string, and so does the
  current binary while `DEVBOX_FEATURE_PKG_DEFS` is off. Projects that adopt
  package definitions should set `devbox_version` in `devbox.json`.

Install location: `$XDG_CACHE_HOME/devbox/pkgs/<name>/<version>/<system>-<key>/`,
shared across projects that would install the same artifact the same way
(see Cache layout and garbage collection below). Nothing is copied per
project; `PATH` points at the cache. Garbage collection of unused versions is
a follow-up (`devbox cache gc`).

## Security

The trust model equals plugins today: a definition is remote JSON that can
run shell through `hooks.post_install` and `shell.init_hook`. Mitigations:

- Checksums are required on every install driven by a lockfile entry that
  records a digest (see trust on first use above).
- Install prints a notice, the first time a version is installed into the
  shared cache, when a definition runs `post_install` hooks, uses `script`
  types or installs an artifact without a digest. `shell.init_hook` is not
  covered, as for plugins.
- Remote definitions must be `https://`. They may not reference `file://`
  URLs, absolute or relative, anywhere.
- Version strings from version sources must match
  `^[0-9A-Za-z][0-9A-Za-z.+_-]*$`; others are ignored. Versions become path
  components and template input.
- Archives are extracted with every entry, including through symlinks
  created by earlier entries, confined to the install directory.
- The search service only returns curated definitions.

## Roadmap: Nix-optional Devbox

The long-term goal is that a project which uses no nixpkgs packages does not
need Nix at all. Package definitions are the first step. This section records
the audit of the stack against that goal and the design for the follow-ups.
None of it is implemented in this stack. Each subsection ends with the work
it implies.

Motivating case: a React Native / Expo iOS project with `nodejs@22`, `jdk17`
and `ccache` from nixpkgs. Nix's `mkShell` stdenv puts clang-wrapper and the
Apple SDK on `PATH`, and exports `CC`, `CXX`, `LD=ld`, `AR`, ...,
`NIX_CFLAGS_COMPILE`, `NIX_LDFLAGS`, `SDKROOT`, `DEVELOPER_DIR` and
`MACOSX_DEPLOYMENT_TARGET`. Native `xcodebuild` then picks up Nix's libc++
headers and feeds clang flags to raw `ld`. The project works around it by
overriding about 20 env vars in `devbox.json`. With `pkg:nodejs` and
`pkg:java`, and no stdenv, none of those overrides would be needed.

### Nix is opt-in

Today every pkg:-only project, and every project with zero packages, still
needs Nix:

- `ensureNixInstalled` is the `PreRunE` of `add`, `shell`, `run`, `install`,
  `shellenv`, `rm`, `list`, `info`, `update`, `services`, `generate` and
  more (`internal/boxcli/setup.go`). `devbox global` goes through the same
  commands. Only `devbox init` skips it.
- `recomputeState` always generates a flake that uses `pkgs.mkShell`. It
  also resolves the nixpkgs stdenv into `devbox.lock` (`nix flake
  metadata`), runs `nix print-dev-env` and syncs the nix profile. This
  happens even when there are no packages.
- `computeEnv` merges the `print-dev-env` output, so the whole stdenv lands
  in the shell. It also always adds `.devbox/nix/profile/default/bin` to
  `PATH`, and sets `DEVBOX_PACKAGES_DIR` and `XDG_DATA_DIRS` from the
  profile.
- `nix.System()` shells out to `nix` and returns `""` without it. Package
  definitions depend on it in two places:
  - `installPkgDef` fails with "does not support".
  - `GetPkgDefConfigsForPackages` logs the plugin error at debug level and
    drops the definition, so its env and init hook are silently lost.
  - `IsEnabledOnPlatform` has the same dependency.
- `--pure` requires `nix` on `PATH` and builds bash from nixpkgs.
- `devbox update` always refreshes the stdenv.
- `devbox services` adds `process-compose` from nixpkgs.

Design:

- **Nix mode.** A project is in Nix mode when its resolved package set
  contains a nixpkgs or flake package. The set includes top-level packages,
  includes, plugin `packages`, and is filtered to the current platform.
  `pkg:` and `runx:` packages are not Nix packages. It is computed after
  config load.
- **Outside Nix mode** Devbox does not require Nix:
  - No install check, no generated flake, no stdenv lock entry (an existing
    one is kept, not added), no `print-dev-env` and no nix profile.
  - Scripts and plugin files are still written, and the state hash is still
    saved.
  - The env is the user env, `devbox.json`, plugin and definition env,
    `DEVBOX_*` vars, and the `bin` dirs of `pkg:` and `runx:` packages. No
    stdenv vars and no profile paths.
- **`--pure` without Nix** keeps package `bin` dirs, `.devbox/bin` and
  `/usr/bin:/bin`. The shell is `$SHELL`, falling back to `/bin/bash`.
- **System detection** comes from `GOOS`/`GOARCH` (the mapping already exists
  in `internal/nix/install.go`), never from `nix`. This fixes the silent loss
  of definition env on machines without Nix. Done for package definitions in
  PR 3 (`nix.SystemOrHost`); `IsEnabledOnPlatform` still calls `nix`.
- **Lazy install.** Nix is installed the first time a Nix package is needed:
  - `devbox add <nix package>` validates the package.
  - Or `ensureStateIsUpToDate` finds the project has entered Nix mode (a
    hand-edited config, or an include or plugin that adds a Nix package).
  - On a TTY Devbox asks "X comes from nixpkgs and needs Nix. Install it?".
    Otherwise it errors with the install hint.
- **`devbox global`** follows the same rule, so `eval "$(devbox global
  shellenv)"` in an rc file never forces a Nix install for an empty or
  pkg:-only global profile.
- **Updates and services.** `devbox update` refreshes nixpkgs only in Nix
  mode (or when asked by name). `devbox services` uses a `process-compose`
  package definition.
- **Nix packages without the stdenv.** A Nix-mode project can set
  `"nix": {"stdenv": false}`. It keeps the nix profile on `PATH` and skips
  the `print-dev-env` env, as `devbox global` and the hidden
  `--omit-nix-env` flag already do. This lets a project keep a nixpkgs tool
  that has no definition yet (e.g. `ccache`) without the toolchain env. An
  alternative is generating the flake with `mkShellNoCC`.
- **PATH order** (done in PR 3). Definition `bin` dirs used to be appended
  after the Nix `PATH` and after `env.PATH` from `devbox.json`, so they lost
  to a nix package or a system binary of the same name. They now come
  first, ahead of the nix profile and the inherited `PATH`. `env.PATH` in
  `devbox.json` still applies on top: a project that prepends `/usr/bin` on
  purpose (as the iOS project above does to escape the stdenv) still gets
  `/usr/bin/java` first, and should drop that override once it no longer
  uses the stdenv.
- **Switching modes.** Leaving Nix mode leaves `.devbox/gen/flake` and the
  profile stale. Delete them or ignore them.

Work: system detection without Nix; a `needsNix()` gate in `Devbox` that
replaces the `PreRunE`; the Nix-free branches of `recomputeState`,
`computeEnv`, `--pure`, `update`; `"nix": {"stdenv": false}`; definitions
for `process-compose` and `ccache`.

### `devbox add go`: shipped definitions and bare names

Gaps in the stack today:

- `pkg:go@1.22` already has a meaning. `ParseRef` reads it as the relative
  file `./go` and fails with "file does not exist", or "is a directory" in a
  Go repo. This namespace should be reserved before the flag is removed.
- A ref with `@` in its source and no version splits wrongly. For example
  `pkg:https://user@host/x.pkg.json` gets version `host/x.pkg.json`, and
  `Versioned()` then never appends `@latest`. A version must not contain
  `/`.
- Path sources are not normalized. `pkg:go.pkg.jsonc` and
  `pkg:./go.pkg.jsonc` are different packages, so both are installed, and
  both map to the same `pkgdef-<name>` virtenv.
- `devbox rm go` does not match `pkg:./go.pkg.jsonc@1.22`, and adding a
  `pkg:` package never replaces a nixpkgs `go@x`.
- `devbox add` stores `--outputs`, `--allow-insecure` and `--patch` for
  `pkg:` packages, where they do nothing. `--disable-plugin` is stored but
  ignored.

Design:

- **Registry refs.** `pkg:<name>@<version>` refers to a shipped definition
  (`pkgs/<name>/<name>.pkg.jsonc`) when `<name>` matches
  `^[a-z0-9][a-z0-9_.-]*$`, has no `/` and no `.pkg.json(c)` suffix. Local
  files must be written `./<file>.pkg.jsonc`, and `ParseRef` rejects a bare
  name that is not in the registry.
  - Shipped definitions are embedded in the binary. The lock entry adds
    `definition_url`: the file at the release tag that produced it,
    alongside `definition_hash`.
  - Loading tries, in order:
    1. the embedded copy, if its hash matches `definition_hash`;
    2. the content cache;
    3. fetching `definition_url` and verifying the hash.
  - So any Devbox version reproduces the same lock, and `devbox update`
    re-pins to the running binary's copy.
  - Full URLs are not written to `devbox.json`. They are noisy, `main` URLs
    are not pinned, and they make `devbox rm go` awkward.
- **Aliases.** `pkgs/index.json` maps nix names to registry names:
  `python3` → `python`, `nodejs_<N>` → `nodejs@<N>`, `jdk<N>` → `java@<N>`.
  An alias that implies a version cannot be combined with `@version`.
- **Choosing nix or native for a bare name.** This only applies when
  `devbox add <name>[@<version>]` names a package with a shipped definition,
  directly or by alias. Other names resolve through search as before.
  - Precedence, highest first:
    1. `--native` / `--nix` flag
    2. `"package_source": "nix" | "native"` in `devbox.json`
    3. `package_source` in `$XDG_CONFIG_HOME/devbox/config.json` (Devbox
       has no user settings file yet)
    4. an interactive prompt
    5. `nix`, until native becomes the default
  - The prompt is shown only when stdin and stderr are terminals. It offers
    to save the answer to `devbox.json` (the default) or to the user config.
    In CI Devbox never prompts. It uses the default and prints a one-line
    hint about `--native`.
  - `devbox.json` always records an unambiguous string: `pkg:go@1.22` or
    `go@1.22`. `package_source` only affects how `devbox add` expands a bare
    name. `install`, `shell`, `run` and `update` never read it, so
    teammates reproduce the same packages without being prompted.
- **No silent fallback.** If native was chosen but the version does not
  resolve, or the system has no artifact, Devbox errors. The error names
  the available versions and platforms and suggests `--nix` or
  `--exclude-platform`. An interactive session may offer nixpkgs for that one
  add, without saving the choice.
- **Identity.** `pkg:<name>`, its nix name and its aliases are one package.
  - `devbox add` replaces the other source, with a notice ("Replacing
    go@1.21 (nixpkgs) with pkg:go@1.22 (native)").
  - `devbox rm go` removes either. If both are present, it asks for the
    full string.
  - Path and URL refs still match only by ref.
- **Flags.** For `pkg:` packages, `devbox add` errors on `--outputs`,
  `--allow-insecure`, `--patch`, `--patch-glibc` and `--disable-plugin`. A
  definition's activation is part of the package. `--platform` and
  `--exclude-platform` stay.
- **Commands.**
  - `devbox info` loads the definition and prints name, resolved version,
    description, homepage and platforms. Today it sends the `pkg:` string
    to the search service and reports "not found".
  - `devbox list` marks native packages.
  - `devbox.schema.json` documents `pkg:` and `package_source`.
  - Builtin plugins match on canonical name, so none fire for `pkg:`
    packages. Registry names should match builtins (e.g. `nodejs`) where
    the plugin still makes sense.

Work: reserve the registry namespace and fix ref parsing (`/` in version,
path normalization) before the flag is removed; embedded registry and
`definition_url`; aliases; `package_source` and the prompt; identity rules
for add/rm; flag validation; `devbox info`.

### Search returns nix and native packages

The search client (`internal/searcher`, `https://www.nixsearch.com`,
`/v2/search`, `/v1/resolve`, `/v2/resolve`) is untouched by this stack. The
service is not in this repo.

Gaps:

- `devbox search pkg:./go.pkg.jsonc@1.22` sends the local path to the
  search service and always gets "No results".
- `devbox list --outdated` resolves package definitions with
  `FetchResolvedPackage`, which runs `Lock`. That fetches checksums for
  every platform just to compare a version, and spends GitHub API quota.
- Multi-project lock sync (`boxcli/multi/sync.go`) only compares
  `last_modified`, which `pkg-def` entries never set.

Options considered:

- **(a) The search service indexes definitions.** One request, no vendor
  calls at query time.
- **(b) The CLI merges results itself.** It would hit 16 vendors (7 on
  GitHub, 60 requests an hour without a token) on every search, and fail
  offline. Rejected.
- **(c) CI publishes a static index.** The weekly `pkgdef-tests` job already
  resolves every definition.

Recommendation: (c) feeding (a).

- CI (`pkgdef-tests`, weekly and on `pkgs/**` changes) writes
  `pkgs/index.json`. Per package it records name, description, homepage,
  registry ref, `definition_hash` and aliases. Per release it records the
  version and the systems `Lock` found an artifact for.
- The search service ingests the index.
  - `/v2/search` results gain `source: "nix" | "native"` (`nix` when absent)
    and a `ref` for native.
  - `/v2/resolve` gains an optional `source` parameter. A native response
    carries `ref`, `definition_hash` and per-system `url` / `sha256` /
    `sha512`.
  - Older CLIs ignore the new fields and see only nix results.
- `devbox search go` shows a `Source` column. A native row's package cell
  is the exact string to pass to `devbox add`.
- `devbox search go@1.22` prints one line per source.
- Ranking is server-side. Within one name, nix comes first until native
  becomes the default.
- The index is only for discovery. `devbox add` always re-resolves through
  the definition, so what is installed is what the definition says.
- `devbox search pkg:...` and `devbox info pkg:...` never contact the
  service. They load the definition and call `ResolveVersion`.
- `devbox list --outdated` uses `ResolveVersion` (no `Lock`) for
  definitions. Multi-project sync compares `version` and `definition_hash`.

Work: the index generator in CI; `Source`/`Ref` on the searcher types;
`devbox search` output; local handling of `pkg:` in `search` and `info`;
cheaper outdated; the sync fix. Server: ingest the index and add the fields.

### Cache layout and garbage collection

Today installs go to `$XDG_CACHE_HOME/devbox/pkgs/<name>/<version>/<system>/`.
That key leaves out the definition's identity:

- A fingerprint mismatch `RemoveAll`s the directory, under any shell that
  is using it.
- Two projects with the same name and version but different definitions
  reinstall over each other on every `devbox shell`, because `computeEnv`
  installs each time.
- The same fingerprint with different locked digests (a vendor respin, or
  different trust-on-first-use results) hard-fails the other project.
- The marker's URL is never compared to the locked URL.

Other gaps:

- Downloads are keyed by file name, not URL, so without a digest a stale
  file from another definition is reused.
- The archive is kept when extraction or a hook fails. `*.tmp-*`, `*.part-*`
  and empty download dirs leak after a kill.
- `.definitions/` grows without bound.
- rust's shared `RUSTUP_HOME` (`<cache>/rustup`) is written by every rust
  install without a lock, and toolchains are never removed. A definition
  named `rustup` would install into that same directory.

Devbox has no GC command today. Nix packages rely on each project's
`.devbox/nix/profile/default` being a GC root, and on the user running
`nix store gc`.

Design:

- **Layout** (done in PR 2). Installs go to
  `$XDG_CACHE_HOME/devbox/pkgs/<name>/<version>/<system>-<key>/`, where
  `<key>` is the first 12 hex digits of sha256 over:
  - the install fingerprint;
  - the system;
  - the lockfile's `url` for that system.

  It is computed from the lockfile before downloading. Two projects share
  an install only when they would install the same artifact the same way.
  The digest is not part of the key: trust on first use records it after
  installing, and the directory must not move when it does. A locked digest
  that differs from the one an install was made from is a checksum error.
- **Immutable installs** (done in PR 2). A directory with
  `.devbox-pkgdef.json` is never deleted or replaced by install, only by GC.
- **Locks** (done in PR 2). `<install>.lock` serializes install and GC
  (flock). Lock files are never deleted.
- **Downloads** (done in PR 2) go to `.downloads/<sha256(url)>-<key>/`, one
  directory per install so concurrent installs of the same URL never share
  a file, and are deleted after unpacking, on success or failure.
- **Reserved names.** Names starting with `.` are reserved for Devbox. State
  shared across installs lives only under `.shared/`, behind its own lock.
  A definition otherwise keeps manager state inside `{{ .InstallDir }}`:
  rust sets `RUSTUP_HOME={{ .InstallDir }}/rustup`, so a toolchain lives and
  dies with its install.
- **Roots.** Each activation of a project with `pkg:` packages writes
  `$XDG_STATE_HOME/devbox/roots/<project-hash>.json`. It lists the project
  path, the install dirs and the definition hashes in use, and is rewritten
  only when it changes. Activation also touches
  `<install>/.devbox-pkgdef-used` (at most hourly). `devbox global` is
  covered the same way.
- **`devbox cache gc [--older-than DURATION] [--dry-run] [--all]`** (default
  `30d`). It is user-run only and never automatic.
  - Drops roots whose `devbox.json` no longer exists.
  - Deletes installs no remaining root lists and last used before
    `--older-than` (`--all`: every unlisted install). For each one, it
    takes the lock without blocking and skips the install if the lock is
    held, then renames the dir to `*.gc-*` and deletes it.
  - Removes unreferenced definitions, all downloads, and leftovers older
    than a day (`*.tmp-*`, `*.part-*`, `*.gc-*`, unmarked installs).
  - Prints the space freed and reminds the user about `nix store gc`.
- An install that is collected and needed again is reinstalled from the
  lockfile on the next activation.
- Project-local state (`.venv`, `.devbox/virtenv/pkgdef-*`) belongs to the
  project and is never collected.

Done: the keyed layout, immutable installs and download fixes (PR 2; this
resolves the shared-path review comment on PR 1). Work: per-install
`RUSTUP_HOME`; roots and used-stamps; `devbox cache gc`.

### Audit findings across the stack

This is a correctness audit of PRs 1–6, merged with `main`, done in November 2026. It is ranked by
severity. Each entry names the PR it was fixed in. All twelve are fixed,
with tests.

1. **Extraction escapes the destination through chained symlinks** *Fixed in PR 2.* (PR 2,
   `extract.go`).
   - The symlink target is checked only as a string. `MkdirAll` and
     `OpenFile` then follow links already on disk.
   - A tar with `s1 -> .`, `s1/s2 -> ..` and `s1/s2/evil` writes outside
     the staging dir. Each extra level climbs one more directory.
   - Fix: resolve the parent with `EvalSymlinks` and require it inside
     `dest`, or extract through `os.Root`.
2. **Vendor version strings are not validated** *Fixed in PR 2; `+build` versions in PR 5.* (PR 2, made reachable by
   PR 5's `+build` rule).
   - `99+/../../../tmp/evil` is not a pre-release, so `latest` picks it.
     `InstallDir` then leaves the cache, and install `RemoveAll`s it.
   - `1.24.0+$(cmd)` is rendered into `post_install` (rust's
     `--default-toolchain {{.Version}}`).
   - Fix: ignore versions not matching `^[0-9A-Za-z][0-9A-Za-z.+_-]*$`, and
     check that `InstallDir` stays under the cache.
3. **Remote definitions can read local files** *Fixed in PR 2; lookup and version override URLs in PR 5.* (PR 2).
   - `http://` definition URLs are accepted.
   - A remote definition may use `file://` in `versions`, `lookup` or
     `checksum` URLs. With a jq `select` it can send, for example,
     `~/.docker/config.json` to its own host, without hooks and without a
     warning.
   - Fix: accept only `https://` definitions. Reject every `file://` URL in
     a remote definition. (The spec already says they must be absolute.)
4. **Activation templates fail silently** *Fixed in PR 3.* (PR 3, `plugin/pkgdef.go`).
   - The definition is JSON-marshalled before templating, so
     `{{ if eq .OS "darwin" }}` becomes `\"darwin\"` and fails to parse.
   - The template funcs (`trimPrefix`, ...) are not registered there.
   - The error is logged at debug level, so env, init hook and
     `create_files` just disappear.
   - Fix: render each string field with `pkgdef.Render` before marshalling,
     and surface load and template errors as warnings.
5. **Install dirs are shared across different definitions.** *Fixed in PR 2, wiring in PR 3.* See *Cache
   layout* above.
   - With trust on first use, `installedFromMarker` also accepts another
     definition's install, and its URL and digest are written into this
     project's lockfile.
   - Fix: the keyed layout. An install is only reused, and its digest only
     recorded, when the marker URL equals the locked URL.
6. **Tar hard links extract as empty files** *Fixed in PR 2.* (PR 2). Fix: link or copy the
   earlier entry, with the same containment check.
7. **`@1.20` resolves to `1.20`, not `1.20.14`** *Fixed in PR 2.* (PR 2, hits `pkgs/go`).
   go.dev names `.0` releases `1.20`, `1.19`, ..., and an exact match wins.
   Fix: put the exact match in the candidate set and take the highest
   prefix match.
8. **Unversioned `pkg:` entries never update** *Fixed in PR 3.* (PR 3, `update.go`).
   `shouldUpdateLockEntry` skips them, yet `ErrDefinitionChanged` tells the
   user to run `devbox update`. Fix: always update `pkg:` entries.
9. **`@` in a ref without a version** *Fixed in PR 2.* (PR 2, `ref.go`). For example
   `pkg:https://cdn/npm/@acme/tool.pkg.json`. See *`devbox add go`* above.
   Fix: split only when the last `@` comes after the last `/`.
10. **`pkg:` reaches nix code paths** *Fixed in PR 3.* (PR 3).
    - `devbox info` and `devbox search` send it to the search service.
    - With the flag off, `devbox add pkg:./x.pkg.jsonc` falls into the slow
      `nix.Search` fallback, because the canonical name contains `.`.
      Existing lock entries later fail with `unsupported flake reference
      URL scheme: pkg`.
    - Fix: with the flag off, fail early with "set
      DEVBOX_FEATURE_PKG_DEFS=1".
11. **PATH order** *Fixed in PR 3.* (PR 3). See *Nix is opt-in* above. The
    order is now: definition `bin` dirs, the nix profile, then the
    `print-dev-env` and inherited `PATH`. `env.PATH` from `devbox.json` is
    applied on top, and runx is still appended last.
12. **CI path filter** *Fixed in PR 6.* (PR 6). `pkgdef-tests.yaml` misses the wiring files:
    `devbox/packages.go`, `devbox.go`, `update.go`, `lock/`, `devpkg/` and
    `devconfig/`.

The corrections to this document that the audit found are applied in the
sections above (lockfile hash format, trust on first use, the install
notice, `devbox info`, the feature flag, and the security rules).

## Implementation plan

### PR 1: spec and Go example (this document)

- `specs/package-definitions.md`
- `pkgs/go/go.pkg.jsonc`

### PR 2: `internal/pkgdef`

A leaf package with no dependency on `devpkg`, `lock` or `plugin`.

- `Parse` and `Validate` for the format above (hujson input).
- `ParseRef` for `pkg:<ref>@<version>`, accepting `.pkg.json` and
  `.pkg.jsonc` refs.
- `Load` for path, `https://` and `file://` refs, with a content-addressed
  cache for remote definitions.
- `ResolveVersion` for `static`, `http_json` and `github_releases`.
- `Lock` that renders the artifact URL and fetches the checksum for every
  supported system.
- `Install` that downloads once, verifies, extracts or places the binary, and
  runs `post_install`. Supports `file://` so testscripts stay offline.
- Unit tests with `httptest`.

### PR 3: wiring, lockfile, env, plugins, testscripts

Behind the `DEVBOX_FEATURE_PKG_DEFS` flag (on in dev builds).

- `pkgtype.IsPkgDef`; excluded from flake detection like runx.
- `devpkg.Package`: `IsPkgDef`, `IsNix` false so shellgen skips it,
  `ValidateExists` resolves through the lockfile.
- `lock.FetchResolvedPackage` branch producing the entry above; new
  `omitempty` fields on `lock.Package` and `lock.SystemInfo`.
- `Devbox.installPackages` installs package definitions next to runx;
  `computeEnv` adds their `bin` directories to `PATH`.
- `devconfig.loadRecursive` creates a plugin config per package definition so
  `env`, `shell.init_hook` and `create_files` apply.
- Local definition file hashes feed the config hash so edits trigger
  reinstall.
- `devbox update` re-resolves through the same lockfile branch.
- Testscripts under `testscripts/pkgdef/` using `file://` artifacts.

### Later

- Search service: `/v2/resolve` returns `source: "pkg-def"` plus the
  definition and per-system `url`/`sha256`; `devbox.json` keeps `go@1.22`.
  Per-package opt out via `"source": "nixpkgs"`.
- Authoring tooling: `devbox pkgdef validate`, `devbox pkgdef new` with an
  LLM skill, a `jetify-com/devbox-packages` repo with a CI matrix, and a
  converter for the aqua registry as a seed corpus.
- `manager` and `script` strategies, `github:` refs, `devbox cache gc`,
  artifact mirroring.

## Risks and open questions

- Linux vendor binaries assume glibc and FHS paths. They will not run on
  NixOS or musl systems; document it and keep nixpkgs as the fallback.
- No binary cache: if a vendor deletes an artifact, a locked project breaks
  until Jetify mirrors artifacts. The lockfile `url` field makes that a
  service-side change.
- `devbox add pkg:./go.pkg.jsonc` does not replace an existing nixpkgs `go@x`
  entry today because canonical names differ (`pkg:./go.pkg.jsonc` vs `go`).
  Decide whether it should once search returns definitions.
- `devbox global` shares `computeEnv` and should work unchanged; verify.
- Windows and `devbox generate` images are out of scope for the first cut.

## Status

| Item | Status |
|---|---|
| Spec and Go example | in review (PR 1) |
| `internal/pkgdef` library | in review (PR 2) |
| Wiring behind feature flag | in review (PR 3) |
| `manager` strategy | not started |
| `script` strategy | not started |
| `github:` refs | not started |
| Search service integration | not started |
| Authoring tooling | not started |
| Cache GC | not started |
