# Package definitions: installing packages directly from the vendor

Status: in progress. This document is both the format specification and the
living implementation plan. Update the status table at the bottom as work
lands.

## Goal

Let a Devbox project install a tool from the upstream vendor's own binary
distribution (go.dev, nodejs.org, GitHub releases, ...) instead of nixpkgs,
driven by a small declarative *package definition* file. Go was the first
package (see [`pkgs/go/go.pkg.jsonc`](../pkgs/go/go.pkg.jsonc)); the initial
set of 16 is listed under [Shipped packages](#shipped-packages). Later, the
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

Inside a definition, a relative `file://` URL (`file://vendor/x.tar.gz`) is
resolved against the directory of a local definition file when fetching.
The lockfile records the URL as rendered, so a project that ships artifacts
next to its definition stays portable. Remote definitions must use absolute
URLs.

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
| `description`, `homepage`, `license` | no | Informational. `homepage` is shown by `devbox info`. |
| `platforms` | yes | Non-empty list of nix system names the definition supports. |
| `os_map`, `arch_map` | no | Overrides for the `{{.OS}}` and `{{.Arch}}` template variables. Keys are the nix OS (`darwin`, `linux`) and nix arch (`x86_64`, `aarch64`). |
| `versions` | yes | A version source, or a list of them. See below. |
| `install` | yes | An install strategy. See below. |
| `bin` | no | Directories relative to the install root that go on `PATH`. Default `["bin"]`. |
| `binaries` | no | Executable names expected in a `bin` directory after install. Install marks them executable (npm tarballs drop the bit) and fails if one is missing. |
| `env`, `shell`, `create_files` | no | Plugin-compatible activation. See Activation. |
| `hooks.post_install` | no | Shell commands run once per installed version by `/bin/sh` in the install root, with `DEVBOX_PKG_INSTALL_DIR`, `DEVBOX_PKG_VERSION` and `DEVBOX_PKG_SYSTEM` set and `/usr/bin:/bin` appended to `PATH`. |

### Version sources (`versions`)

| `type` | Fields | Behaviour |
|---|---|---|
| `static` | `versions: [string]` | The list is the universe of versions. |
| `http_json` | `url`, `select` | GET `url`, parse JSON, evaluate the jq expression in `select`. Every string (or number) it emits is a version. When `url` contains `{{.Page}}`, pages 0, 1, ... are fetched until one returns 404 or no versions. |
| `github_releases` | `repo` (`owner/name`), `strip_prefix` (default `"v"`) | List releases through the GitHub API, 100 per page, skip drafts and pre-releases, strip the prefix from tag names. Honors `GITHUB_TOKEN`. |
| `script` | `run` | Planned. Run `run` with `sh -c`; one version per output line. |

`versions` may also be a list of sources, whose versions are combined. Yarn
needs this: 1.x is the `yarn` npm package and 2.x onward is
`@yarnpkg/cli-dist`.

Paginated sources (`github_releases`, and `http_json` with `{{.Page}}`) are
read lazily: resolution stops at the first page that has a version matching
the request, so `latest` costs one request and `kubectl@1.30` reads as many
pages as it takes to reach 1.30. Sources should list newest versions first.
At most 20 pages are read.

Version strings are compared as dotted numeric segments, falling back to
string comparison. Build metadata after `+` (OpenJDK's `21.0.5+11`) only
breaks ties, so a vendor respin (`21.0.12+101`) sorts after the original
build (`21.0.12+8`). Versions whose part before `+` contains a hyphen or a
letter (`1.23.0-rc1`, `3.14.0a1`) are pre-releases and only match when
requested exactly.

Matching a requested version against the list:

| Requested | Match |
|---|---|
| `latest` | The highest non-pre-release version. |
| `1.22` | The highest version equal to `1.22` or starting with `1.22.`. |
| `1.22.3` | That version exactly (also by the prefix rule, so `1.22.3` matches `1.22.3.1`-style versions if the exact one is absent). |
| `21.0.5` | The highest `21.0.5+<build>`. |

### Install strategies (`install`)

`install.type` picks how a version lands on disk.

| `type` | Fields | Behaviour |
|---|---|---|
| `archive` | `url` and `checksum`, or `lookup`; `strip_components` (default `0`) | Download the archive, verify it, extract it into the install root. Supports `.tar.gz`, `.tgz`, `.tar.xz`, `.tar.bz2`, `.tar.zst` and `.zip`. |
| `binary` | `url` and `checksum`, or `lookup`; `bin_name` (default: `name`) | Download a single executable to `<root>/bin/<bin_name>` and mark it executable. |
| `manager` | `requires`, `env`, `install`, `locate` | Planned. Delegate to another definition that owns version management (see below). |
| `script` | `run` | Planned. Escape hatch: run `run` with the install root as `$DEVBOX_PKG_INSTALL_DIR`. |

#### Looking up the artifact in an index (`install.lookup`)

Some vendors' download names can't be derived from the version:
python-build-standalone puts a build date in every file name, Temurin renames
respins (`21.0.12+101` ships as `..._21.0.12.1_1.tar.gz`), and zig renamed
its tarballs in 0.14. Their indexes list the exact URL and checksum instead.
`lookup` replaces `url` and `checksum`:

```jsonc
"install": {
  "type": "archive",
  "lookup": {
    "url": "https://ziglang.org/download/index.json",
    "select": ".[\"{{.Version}}\"][\"{{.Arch}}-{{.OS}}\"] // empty | {url: .tarball, checksum: .shasum}"
  }
}
```

`url` and `select` are templates. `select` is a jq expression that must
produce an object with `url` and, optionally, `checksum` (any digest format
below). If it produces nothing, or `url` returns 404, the vendor has no
artifact of that version for that system.

#### Overrides

`install.overrides` maps a nix system to a partial `install` object whose
fields replace the defaults for that system (for example a different
`strip_components` for macOS bundles).

`install.version_overrides` is a list of partial `install` objects for vendors
that changed how they ship artifacts. Each has a `versions` constraint: one or
more comparisons with `<`, `<=`, `>`, `>=`, `=` or `!=`, separated by spaces
or commas, that must all hold (`"<11"`, `">=2, <4"`). A bare version matches
itself and every more specific version. The first entry that matches applies.
An entry may have its own `overrides` per system but no nested
`version_overrides`.

```jsonc
"install": {
  "type": "archive",
  "url": "https://github.com/pnpm/pnpm/releases/download/v{{.Version}}/pnpm-{{.OS}}-{{.Arch}}.tar.gz",
  "checksum": { "type": "github_release", "repo": "pnpm/pnpm" },
  "version_overrides": [
    {
      "versions": "<11",
      "url": "https://registry.npmjs.org/@pnpm/{{ .OS | replace \"darwin\" \"macos\" }}-{{.Arch}}/-/...",
      "checksum": { "type": "http_json", "url": "...", "select": ".dist.integrity" },
      "strip_components": 1
    }
  ]
}
```

Overrides apply in this order: the matching version override, the
definition's override for the system, then the version override's own
override for the system. `url` and `lookup` are alternatives, so an override
that sets one clears the other.

Why two real strategies: `archive` covers more than it first appears. Go,
nodejs, deno, bun, zig, Temurin JDK and python (via python-build-standalone,
the same builds uv installs) all ship per-platform archives, so fnm, nvm and
pyenv are not needed for them. `manager` is for ecosystems where the vendor
ships no usable archive or the manager is the ecosystem's contract: rustup
for rust, ruby (no official binaries), php on Linux.

Until `manager` exists, the shipped `rust` definition gets the same result
with `binary` plus a `post_install` hook: it installs the checksummed
`rustup-init` and runs it to install the requested toolchain into a shared
`RUSTUP_HOME`, and its `env` sets `RUSTUP_TOOLCHAIN` to the version.

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
| `http_json` | `url`, `select` | GET `url`, evaluate the jq expression; the first string result is the digest. Covers JSON indexes and the npm registry's `dist.integrity`. |
| `sha256_url` | `url` | GET `url`. The body is either a bare hex digest or a `sha256sum`-style list (`<hex>  <filename>` per line) matched by `{{.Filename}}`. Covers nodejs `SHASUMS256.txt` and the common `<asset>.sha256` convention. |
| `sha512_url` | `url` | Like `sha256_url` for sha512 digests (Apache's `.sha512` files). |
| `github_release` | `repo`, `tag` (default `v{{.Version}}`) | The digest GitHub computes for every release asset, read through the GitHub API (honors `GITHUB_TOKEN`). Assets uploaded before GitHub started computing digests (mid 2025) have none, which is treated like `none`. |
| `none` | | No vendor-published checksum. `devbox add` records the digest it observes for the current system (trust on first use). Other systems are filled in lazily when a machine of that system installs, matching how nix store paths are backfilled today. |

Digests may be sha256 or sha512, as hex (the algorithm follows from the
length), as `sha256:<hex>`/`sha512:<hex>`, or as an SRI string
(`sha512-<base64>`).

Checksums are mandatory when installing from an existing lockfile entry: the
downloaded artifact must match the digest recorded for the system, or the
install fails.

#### Platforms a version is not published for

Vendors add and drop platforms over time: Node.js had no Apple silicon
build before 16, Java 8 has none at all, Temurin 27 has no Intel macOS
build. When the checksum source shows that an artifact does not exist (the
checksum file or lookup index returns 404, a checksum list has no entry for
the file, a jq expression matches nothing, the GitHub release has no such
asset), that system is left out of the lock entry. Locking fails only when no
platform has an artifact. Installing on a system that the lock entry leaves
out fails with an error that names the platforms the version is available
for. Definitions with checksum `none` cannot tell, so their lock entries
list every platform.

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
| `{{.Page}}` | `versions.url` (`http_json`) | The 0-based page number. See Version sources. |

Functions: `trimPrefix`, `trimSuffix`, `replace`, `lower`, `upper`,
`splitList`, plus the `text/template` builtins (`eq`, `index`, `urlquery`,
...). Arguments follow the sprig order with the string last, so pipelines read
naturally: `{{ .Version | trimPrefix "v" }}`, `{{ replace "." "_" .Version }}`,
`{{ index (splitList "." .Version) 0 }}` for the major version.

### Activation

`env`, `shell.init_hook` and `create_files` have the same schema as in
`plugin.json`. Devbox turns each package definition into a plugin whose source
is the definition, so init hooks and created files work through the existing
plugin manager with no new machinery. `create_files` sources are resolved
relative to the definition file; they are only supported for local
definitions. The plugin's `{{ .Virtenv }}` and `{{ .DevboxDir }}` are
namespaced as `pkgdef-<name>` so they never collide with a built-in plugin
or nix package of the same name.

A definition that cannot be loaded (deleted file, unreachable URL, changed
remote content) does not prevent `devbox.json` from loading, so
`devbox update` and `devbox rm` can always run; the install step reports
the error.

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
  "definition_hash": "sha256-...",
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
- `systems.*.url`, `systems.*.sha256` and `systems.*.sha512` are new
  `omitempty` fields on the existing per-system struct. `lockfile_version`
  stays `1`.
- A system missing from `systems` means the vendor does not publish this
  version for it.
- Older Devbox binaries reject the `pkg:` package string. Projects that adopt
  package definitions should set `devbox_version` in `devbox.json`.

Install location: `$XDG_CACHE_HOME/devbox/pkgs/<name>/<version>/<system>/`,
shared across projects. Nothing is copied per project; `PATH` points at the
cache. Garbage collection of unused versions is a follow-up
(`devbox cache gc`).

Each install directory carries a marker with the *install fingerprint* of
the definition: a hash of the install strategy that applies to the version
and system (after overrides), `bin`, `binaries`, `hooks`, `os_map` and
`arch_map`. When a definition's install recipe changes for a version
that is already installed, the next install redoes it. Activation-only
edits (`env`, `shell`, `create_files`) do not trigger a reinstall; they are
applied at shell time.

Trust-on-first-use digests are written to `devbox.lock` only by the install
step (`devbox add`, `devbox install`, `devbox update`), never while
computing the environment for `devbox shell` or `devbox shellenv`.

## Shipped packages

Definitions live in `pkgs/<name>/<name>.pkg.jsonc`, each next to a
testscript (`<name>.test.txt`) that installs two versions and switches
between them. All of them support the four platforms; a version a vendor does
not publish for a platform is skipped for it (see above).

| Package | Source | Versions from | Checksum | Notes |
|---|---|---|---|---|
| `go` | go.dev | go.dev JSON index | index (`http_json`) | Sets `GOTOOLCHAIN=local` like the nixpkgs go, so the pinned version runs even when go.mod asks for a newer one. |
| `nodejs` | nodejs.org | `index.json` | `SHASUMS256.txt` | Includes npm (and corepack before 25). `npm install -g` goes to the project's virtenv. |
| `pnpm` | GitHub releases (11+), npm `@pnpm/<os>-<arch>` (<11) | GitHub releases | GitHub asset digest; npm sha512 integrity | Standalone executable with its own Node.js. `version_overrides` for the layout change in 11. |
| `yarn` | npm `@yarnpkg/cli-dist` (2+), `yarn` (1.x) | both npm packages (list of sources) | npm sha512 integrity | Needs `nodejs`. `binaries` restores the executable bit npm drops. |
| `python` | python-build-standalone | uv's `download-metadata.json` | index (`lookup`) | Creates and activates `$VENV_DIR` (default `.venv`), recreating it when the version changes, so pip never writes to the shared install. |
| `uv` | GitHub releases | GitHub releases | `<asset>.sha256` | |
| `rust` | static.rust-lang.org (rustup-init) | rust-lang/rust GitHub releases | `rustup-init.sha256`, then rustup's manifest | `post_install` runs rustup-init; toolchains share `RUSTUP_HOME` in the cache; `RUSTUP_TOOLCHAIN` selects the version. |
| `java` | Adoptium (Eclipse Temurin JDK) | Adoptium API, paged | index (`lookup`) | Versions are `21.0.5+11`. `JAVA_HOME` is set. macOS builds are bundles, hence a per-system `strip_components`. |
| `gradle` | services.gradle.org | `versions/all` | index (`http_json`) | Needs `java`. |
| `maven` | Maven Central | Maven Central search API | Apache archive `.sha512` | Needs `java`. Central has digests for only some releases. |
| `deno` | GitHub releases | GitHub releases | `<asset>.sha256sum` | Releases before 2.0.1 have no checksum (trust on first use). |
| `bun` | GitHub releases | GitHub releases | `SHASUMS256.txt` | `post_install` links `bunx`. |
| `zig` | ziglang.org | `index.json` | index (`lookup`) | |
| `terraform` | releases.hashicorp.com | `index.json` | `SHA256SUMS` | |
| `kubectl` | dl.k8s.io | kubernetes/kubernetes GitHub releases | `<binary>.sha256` | |
| `helm` | get.helm.sh | helm/helm GitHub releases | `<asset>.sha256sum` | |

Not shipped yet: ruby and php (no official binaries; `manager`), dotnet,
erlang/elixir, and tools that are a one-line `binary` definition each (gh,
jq, just, ...).

## Security

The trust model equals plugins today: a definition is remote JSON that can
run shell through `hooks.post_install` and `shell.init_hook`. Mitigations:

- Checksums are required on every install driven by a lockfile.
- `devbox add` prints a notice when a definition uses `script` types or
  `checksum: none`.
- The search service only returns curated definitions.

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

Behind the `DEVBOX_FEATURE_PKG_DEFS` flag (on in dev builds). While the
flag is off, `pkg:` strings are not recognized at all.

- `pkgtype.IsPkgDef`; excluded from flake detection like runx.
- `devpkg.Package`: `IsPkgDef`, `IsNix` false so shellgen skips it,
  `ValidateExists` resolves through the lockfile.
- `lock.FetchResolvedPackage` branch producing the entry above; new
  `omitempty` fields on `lock.Package` and `lock.SystemInfo`.
- `Devbox.installPackages` installs package definitions next to runx;
  `computeEnv` adds their `bin` directories to `PATH`.
- `devconfig.loadRecursive` creates a plugin config per package definition so
  `env`, `shell.init_hook` and `create_files` apply, with the definition's
  template variables available in both. The plugin's virtenv directory is
  `pkgdef-<name>`. Definitions excluded on the current platform get no
  plugin config.
- Local definition file hashes feed the config hash so edits trigger
  reinstall.
- `devbox update` re-resolves through the same lockfile branch and records
  a changed `definition_hash` even when the version and artifacts are the
  same.
- Testscripts under `testscripts/pkgdef/` using relative `file://`
  artifacts next to the definition.
- `devbox add` fails with the definition's own error (unknown version,
  unsupported system, bad checksum) instead of falling back to nixpkgs.
- A definition whose `platforms` exclude the current system fails at
  install time with a hint to use `--exclude-platform`.

### PR 4: initial package set

- `pkgs/<name>/<name>.pkg.jsonc` for the packages in
  [Shipped packages](#shipped-packages).
- This spec: lists of version sources, paginated sources, versions with
  `+build`, `install.lookup`, `install.version_overrides`, `sha512_url` and
  `github_release` checksums, sha512 and SRI digests, platforms a version is
  not published for, and `binaries` checks.

### PR 5: support for the initial package set

- `internal/pkgdef`: everything listed under PR 4, plus memoized metadata
  requests so locking four platforms fetches an index once, `/bin/sh` and a
  fallback `PATH` for hooks, and an install fingerprint per version.
- `lock.SystemInfo.Sha512`; lock entries omit systems without an artifact,
  and installing on such a system is an error that lists where the version
  is available.
- `ensureStateIsUpToDate` reloads plugin configs after installing, so a
  version changed in devbox.json gets its definition's env and init hook in
  the same run.

### PR 6: tests for the initial package set

- `pkgs/<name>/<name>.test.txt`: install a version from devbox.json, switch
  with `devbox add`, switch back by editing devbox.json, checking the
  version (and env) each time. Run by `TestPackageDefinitions` when
  `DEVBOX_RUN_PKGDEF_TESTS=1`, and by a CI workflow on Linux and macOS when
  `pkgs/` or the implementation changes and weekly to catch vendor changes.
- Every shipped definition parses and is named after its directory (always
  runs); every shipped definition resolves `latest` and locks it against the
  real vendor (network, same flag).

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
- Vendor metadata is a dependency: python's index is a file in uv's
  repository (it moved once already), Java depends on the Adoptium API, and
  Maven on search.maven.org. A broken source fails `devbox add` and
  `devbox update` only; locked projects keep installing from the recorded
  URLs. The weekly CI run is the early warning; the search service could
  serve these indexes later.
- Version lookups through the GitHub API are limited to 60 requests an hour
  without `GITHUB_TOKEN`. Pagination is lazy and responses are memoized, so a
  typical `devbox add` makes one or two requests.
- Hooks run with the user's environment. The rust definition depends on
  rustup-init flags staying compatible; the `manager` strategy would make
  that contract explicit.
- Windows and `devbox generate` images are out of scope for the first cut.

## Status

| Item | Status |
|---|---|
| Spec and Go example | in review (PR 1) |
| `internal/pkgdef` library | in review (PR 2) |
| Wiring behind feature flag | in review (PR 3) |
| Lockfile fields `definition_hash`, `systems.*.url`, `systems.*.sha256` | in review (PR 3) |
| Activation through the plugin manager (`env`, `shell`, `create_files`) | in review (PR 3) |
| Trust-on-first-use checksum recording | in review (PR 3) |
| Initial package set (16 definitions) | in review (PR 4) |
| `lookup`, `version_overrides`, version source lists, pagination, sha512, `github_release` checksums, per-platform availability | in review (PR 5) |
| Per-package install and version-switch tests, CI workflow | in review (PR 6) |
| `manager` strategy | not started |
| `script` strategy | not started |
| `github:` refs | not started |
| Search service integration | not started |
| Authoring tooling | not started |
| Cache GC | not started |
