# Package Search API

Devbox resolves versioned Nixpkgs packages (`go@1.21`, `nodejs@latest`) and
powers `devbox search` by calling a package search service over HTTP. This
document describes the API that Devbox expects that service to implement. Any
server that implements it can be used as a drop-in replacement.

The default host is `https://www.nixsearch.com`. Override it by setting the
`DEVBOX_SEARCH_HOST` environment variable to a base URL:

```sh
DEVBOX_SEARCH_HOST=http://localhost:8080 devbox search ripgrep
```

The client lives in [`internal/searcher`](internal/searcher). The Go types in
[`internal/searcher/model.go`](internal/searcher/model.go) are the source of
truth for the response schemas.

## Endpoints at a glance

| Endpoint          | Used by                                                                                               |
| ----------------- | ----------------------------------------------------------------------------------------------------- |
| `GET /v2/resolve` | Writing `devbox.lock` (`devbox add`, `devbox update`, `devbox install`), `devbox init --auto`         |
| `GET /v1/resolve` | `devbox search <name>@<version>`, `devbox info`, and `devbox.lock` when `DEVBOX_FEATURE_RESOLVE_V2=0` |
| `GET /v1/search`  | `devbox search <query>`                                                                               |

## Conventions

- All requests are `GET` with parameters passed as URL-encoded query
  parameters. Paths are joined onto the host, so the host may include a path
  prefix (for example `https://example.com/devbox`).
- Requests send a `User-Agent` of the form `Devbox/<version> (<os>; <arch>)`,
  for example `Devbox/0.18.3 (darwin; arm64)`.
- Successful responses must be `200` with a JSON body. Unknown fields are
  ignored, so servers are free to return extra data.
- `404 Not Found` means the package or version doesn't exist. Devbox turns
  this into a "package not found" error, and in some flows it is expected
  (auto-detection probes for versions and falls back to `latest` on 404).
- Any other status `>= 400` is treated as a failure. The status line and the
  response body are shown to the user, so a short plain-text body explaining
  the problem is helpful. This includes rate limiting (`429`).

### Version matching

Both resolve endpoints take a package `name` and a `version`, and should return
the single best match:

| `version`      | Expected match                                             |
| -------------- | ---------------------------------------------------------- |
| `latest`       | The newest available version of the package.               |
| `1.21`         | The newest version with that prefix, for example `1.21.13`. |
| `3.1.4`        | That exact version.                                        |

`name` is a Nixpkgs attribute path such as `go`, `python311`,
`nodePackages.pnpm`, or `php82Extensions.redis`. The name is split from the
version at the last `@`, so names can themselves contain `@`.

## `GET /v2/resolve`

Resolves a package version to a Nix flake installable for each supported
system. This is the endpoint Devbox uses to write lock files.

### Parameters

| Name      | Required | Description                               |
| --------- | -------- | ----------------------------------------- |
| `name`    | yes      | Package name (Nixpkgs attribute path).    |
| `version` | yes      | Version constraint (see Version matching). |

### Response

```json
{
  "name": "hello",
  "version": "2.12.3",
  "summary": "Program that produces a familiar, friendly greeting",
  "systems": {
    "aarch64-darwin": {
      "flake_installable": {
        "ref": {
          "type": "github",
          "owner": "NixOS",
          "repo": "nixpkgs",
          "rev": "34ca302a9572963c02e385c056be37c85ff51b77"
        },
        "attr_path": "hello"
      },
      "last_updated": "2026-09-24T08:34:56Z",
      "outputs": [
        {
          "name": "out",
          "path": "/nix/store/mgc3m5ad39b84vkdrca6zd9jan5a28c2-hello-2.12.3",
          "default": true
        }
      ]
    }
  }
}
```

| Field                                  | Required | Description                                                                                                                                                        |
| -------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `name`                                 | yes      | Canonical package name.                                                                                                                                            |
| `version`                              | yes      | The resolved version. Written to `devbox.lock` as `version`.                                                                                                       |
| `summary`                              | no       | Short package description.                                                                                                                                         |
| `systems`                              | yes      | Map keyed by Nix system (`aarch64-darwin`, `aarch64-linux`, `x86_64-darwin`, `x86_64-linux`). Must contain at least one system on success.                         |
| `systems.*.flake_installable.ref`      | yes      | Flake reference in exploded attribute-set form (`type`, `owner`, `repo`, `rev`, ...). For Nixpkgs this is a `github` ref pinned to a commit `rev`.                 |
| `systems.*.flake_installable.attr_path` | yes     | Attribute path of the package within the flake.                                                                                                                    |
| `systems.*.last_updated`               | yes      | RFC 3339 timestamp of the package's last change. Written to `devbox.lock` as `last_modified`.                                                                      |
| `systems.*.outputs`                    | no       | Nix store outputs for this system. Each entry has `name` (such as `out`, `bin`, `man`), absolute `path`, and `default` (whether Nix installs it by default).      |
| `systems.*.outputs[].nar`              | no       | NAR URL in the binary cache. Accepted but currently unused by Devbox.                                                                                              |

How Devbox uses the response:

- The lock file's `resolved` field is built from the flake installable of the
  current system, falling back to `x86_64-linux` and then any available system.
  For the example above it is
  `github:NixOS/nixpkgs/34ca302a9572963c02e385c056be37c85ff51b77#hello`.
- Each system that has `outputs` gets a `systems` entry in the lock file, which
  lets Devbox fetch prebuilt store paths from the binary cache instead of
  evaluating Nixpkgs. Systems without outputs are omitted from the lock file
  and fall back to the slower evaluation path.
- If a system is missing from `systems` entirely, users on that system get the
  fallback installable, which may fail to build if the package isn't supported
  there. Only omit systems that genuinely aren't supported.

## `GET /v1/resolve`

The legacy resolve endpoint. It takes the same parameters as `/v2/resolve`
and returns the matched version with per-system Nixpkgs metadata.

### Response

```json
{
  "name": "hello",
  "version": "2.12.3",
  "summary": "Program that produces a familiar, friendly greeting",
  "commit_hash": "c27cdad491a991b11ed731760aa2ef8db0cb0410",
  "last_updated": 1787814960,
  "systems": {
    "aarch64-darwin": {
      "system": "aarch64-darwin",
      "commit_hash": "c27cdad491a991b11ed731760aa2ef8db0cb0410",
      "last_updated": 1787814960,
      "version": "2.12.3",
      "attr_paths": ["hello"],
      "store_hash": "85py0qgpd9llilbkgjpcwc4svrx056ld",
      "store_name": "hello",
      "store_version": "2.12.3",
      "meta_name": "hello-2.12.3",
      "meta_version": [""],
      "summary": "Program that produces a familiar, friendly greeting"
    }
  }
}
```

| Field                         | Required | Description                                                                                                                  |
| ----------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `name`                        | yes      | Package name. Printed by `devbox search <name>@<version>` and `devbox info`.                                                 |
| `version`                     | yes      | The resolved version. Printed by `devbox search <name>@<version>` and `devbox info`.                                         |
| `summary`                     | no       | Short package description. Printed by `devbox info`.                                                                         |
| `commit_hash`, `last_updated` | no       | Top-level copies of the per-system fields. Not read by Devbox.                                                               |
| `systems`                     | yes      | Map keyed by Nix system. Must contain at least one system.                                                                   |
| `systems.*.commit_hash`       | yes      | Nixpkgs commit that contains this version.                                                                                   |
| `systems.*.attr_paths`        | yes      | Attribute paths for the package in that commit. The first entry is used, so it must be non-empty.                            |
| `systems.*.last_updated`      | yes      | Unix timestamp (seconds) of the package's last change.                                                                       |
| `systems.*.version`           | yes      | The resolved version for this system.                                                                                        |
| `systems.*.store_hash`        | no       | Hash part of the output store path (the 32 characters after `/nix/store/`). Used to look up the store path in `cache.nixos.org`. |
| `systems.*.store_name`        | no       | Store path name. Systems with an empty `store_hash` or `store_name` are skipped when looking up cached store paths.          |
| `systems.*.system`, `store_version`, `meta_name`, `meta_version`, `summary` | no | Informational. Not read by Devbox.                            |

When `/v1/resolve` is used to write a lock file, `resolved` is
`github:NixOS/nixpkgs/<commit_hash>#<attr_paths[0]>` from the same system
selection described for `/v2/resolve`.

## `GET /v1/search`

Free-text package search, used by `devbox search <query>`.

### Parameters

| Name | Required | Description                                                    |
| ---- | -------- | -------------------------------------------------------------- |
| `q`  | yes      | Search query. Devbox never sends an empty query. |

### Response

```json
{
  "num_results": 14,
  "packages": [
    {
      "name": "hello",
      "num_versions": 5,
      "versions": [
        { "name": "hello", "version": "2.12.3", "summary": "Program that produces a familiar, friendly greeting" },
        { "name": "hello", "version": "2.12.2" }
      ]
    }
  ]
}
```

| Field                        | Required | Description                                                                                            |
| ---------------------------- | -------- | ------------------------------------------------------------------------------------------------------ |
| `num_results`                | yes      | Result count, displayed as `Found <n>+ results`.                                                      |
| `packages`                   | yes      | Matching packages, most relevant first. An empty list prints `No results found`.                      |
| `packages[].name`            | yes      | Package name, as a user would pass it to `devbox add`.                                                 |
| `packages[].num_versions`    | yes      | Total versions available. Used to decide whether to show a `...` after truncated version lists.       |
| `packages[].versions`        | yes      | Available versions, newest first. Entries use the same shape as the `/v1/resolve` response.           |
| `packages[].versions[].version` | yes   | Version string to display. Empty versions are skipped.                                                |

Devbox displays packages and versions in the order returned and doesn't
re-sort them, so ranking is entirely up to the server. By default it shows the
first 10 packages and the first 10 versions of each. `--show-all` shows
everything returned.

## Testing a search service

Point Devbox at the server and exercise each endpoint:

```sh
export DEVBOX_SEARCH_HOST=http://localhost:8080

devbox search ripgrep             # /v1/search
devbox search go@1.21             # /v1/resolve
devbox info hello                 # /v1/resolve
devbox init && devbox add hello   # /v2/resolve, check devbox.lock
DEVBOX_FEATURE_RESOLVE_V2=0 devbox add jq   # /v1/resolve lock file path
devbox search nosuchpackage@1.0   # expects a 404
```
