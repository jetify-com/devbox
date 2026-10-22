# Package definitions: installing packages directly from the vendor

Status: in progress. This document is both the format specification and the
living implementation plan. Update the status table at the bottom as work
lands.

## Goal

Let a Devbox project install a tool from the upstream vendor's own binary
distribution (go.dev, nodejs.org, GitHub releases, ...) instead of nixpkgs,
driven by a small declarative *package definition* file. Go is the first
package (see [`pkgs/go/go.pkg.json`](../pkgs/go/go.pkg.json)). Later, the
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

A definition is a JSON file (hujson, comments and trailing commas allowed)
named `<name>.pkg.json`. A project references it with the `pkg:` scheme:

```sh
devbox add pkg:./go.pkg.json@1.22
devbox add pkg:https://raw.githubusercontent.com/jetify-com/devbox/main/pkgs/go/go.pkg.json@1.22
```

The string is `pkg:<ref>@<version>`. The version is split at the last `@`,
exactly like `name@version` today, so the existing config and lockfile code
paths keep working. `<ref>` is one of:

| Form | Example | Notes |
|---|---|---|
| relative path | `./go.pkg.json`, `../defs/go.pkg.json` | Relative to the directory of `devbox.json`. |
| absolute path | `/opt/defs/go.pkg.json` | |
| `https://` URL | `https://example.com/go.pkg.json` | Content is cached by hash; the lockfile pins the hash. |
| `file://` URL | `file:///tmp/go.pkg.json` | Mainly for tests. |
| `github:` | `github:jetify-com/devbox/pkgs/go/go.pkg.json` | Planned. Sugar for the raw.githubusercontent.com URL. |

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

Checksums are mandatory when installing from an existing lockfile entry: the
downloaded artifact must match the `sha256` recorded for the system, or the
install fails.

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
"pkg:./go.pkg.json@1.22": {
  "source": "pkg-def",
  "version": "1.22.3",
  "resolved": "pkg:./go.pkg.json@1.22.3",
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
- `systems.*.url` and `systems.*.sha256` are new `omitempty` fields on the
  existing per-system struct. `lockfile_version` stays `1`.
- Older Devbox binaries reject the `pkg:` package string. Projects that adopt
  package definitions should set `devbox_version` in `devbox.json`.

Install location: `$XDG_CACHE_HOME/devbox/pkgs/<name>/<version>/<system>/`,
shared across projects. Nothing is copied per project; `PATH` points at the
cache. Garbage collection of unused versions is a follow-up
(`devbox cache gc`).

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
- `pkgs/go/go.pkg.json`

### PR 2: `internal/pkgdef`

A leaf package with no dependency on `devpkg`, `lock` or `plugin`.

- `Parse` and `Validate` for the format above (hujson input).
- `ParseRef` for `pkg:<ref>@<version>`.
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
- `devbox add pkg:./go.pkg.json` does not replace an existing nixpkgs `go@x`
  entry today because canonical names differ (`pkg:./go.pkg.json` vs `go`).
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
