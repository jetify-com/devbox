# `devbox.json` Reference

Full schema is at `.schema/devbox.schema.json` in the devbox repo. This is the practical subset.

## Top-level fields

| Field | Type | Notes |
|---|---|---|
| `$schema` | string | Schema version URL. Optional. |
| `name` | string | Project name. Optional. |
| `description` | string | Human-readable description. Optional. |
| `packages` | array \| object | Packages to install. See below. |
| `env` | object | Extra env vars. Only `$PATH` / `$PWD` expand. |
| `shell.init_hook` | string \| array<string> | Runs on every shell/run entry. |
| `shell.scripts` | object | Named commands; value is string or array<string>. |
| `include` | array<string> | Plugin includes. See below. |
| `env_from` | string | Inherit env from another devbox project path. |

## `packages` — two forms

**Array form** (simple, most common):

```json
"packages": ["go@1.22", "nodejs@20", "postgresql@15"]
```

**Object form** (per-package metadata):

```json
"packages": {
  "go":         "1.22",
  "postgresql": { "version": "15", "excluded_platforms": ["aarch64-darwin"] },
  "ffmpeg":     { "version": "latest", "outputs": ["bin", "lib"], "patch": "always" }
}
```

Per-package options:

- `version`: semver or `latest`.
- `platforms`: only install on these platforms (allowlist).
- `excluded_platforms`: skip on these platforms (denylist). Cannot be combined with `platforms`.
- `outputs`: Nix outputs to install (most packages need only the default). Useful for libraries.
- `patch`: `auto` (default), `always`, or `never`. Controls glibc patching on Linux.
- `glibc_patch`: boolean shorthand for forcing glibc patching.
- `allow_insecure`: mark a CVE-flagged package as allowed. Pair with `devbox add --allow-insecure`.
- `disable_plugin`: skip any built-in plugin registered for this package.

Platform values: `i686-linux`, `aarch64-linux`, `aarch64-darwin`, `x86_64-darwin`, `x86_64-linux`, `armv7l-linux`.

## `env` — environment variables

```json
"env": {
  "GOENV":        "off",
  "PATH":         "$PWD/bin:$PATH",
  "DATABASE_URL": "postgres://localhost/dev"
}
```

Only `$PATH` and `$PWD` are expanded. **No command substitution, no other variable expansion.** Put dynamic logic in `init_hook` instead.

## `shell` — scripts and hooks

```json
"shell": {
  "init_hook": [
    "unset GOROOT GOTOOLCHAIN",
    "export PROJECT_ROOT=$PWD"
  ],
  "scripts": {
    "test":  "go test ./...",
    "build": ["go mod tidy", "go build -o bin/app ./cmd/app"],
    "dev":   "air"
  }
}
```

- `init_hook` runs on every `devbox shell` AND every `devbox run`. Keep it cheap.
- Script values can be a single string or an array of strings executed sequentially, stopping on first non-zero exit.
- Invoke with `devbox run <name>`. Arguments after `--` are passed to the final command.

## `include` — plugins

```json
"include": [
  "plugin:nginx",
  "path:./plugins/my-plugin",
  "github:jetify-com/devbox-plugins#path/to/plugin"
]
```

- `plugin:<name>` — built-in plugin shipped with devbox (see `pkg` directory in the devbox repo).
- `path:<dir>` — local plugin directory containing a `plugin.json`.
- `github:owner/repo[#path]` — plugin from a GitHub repo.

Plugins can contribute packages, env vars, services, and files. See `.schema/devbox-plugin.schema.json` for the plugin schema.

## `env_from`

```json
"env_from": "../shared-env"
```

Inherits env and packages from another devbox project on disk. Useful for monorepos with a shared base environment.

## Full example

```json
{
  "$schema":     "https://raw.githubusercontent.com/jetify-com/devbox/0.17.2/.schema/devbox.schema.json",
  "name":        "my-api",
  "description": "Backend service",
  "packages": [
    "go@1.22",
    "postgresql@15",
    "redis@7"
  ],
  "env": {
    "CGO_ENABLED": "0",
    "PATH":        "$PWD/bin:$PATH"
  },
  "shell": {
    "init_hook": [
      "test -d bin || mkdir bin"
    ],
    "scripts": {
      "build": "go build -o bin/api ./cmd/api",
      "test":  "go test -race ./...",
      "dev":   ["devbox services up -b", "air"]
    }
  },
  "include": ["plugin:postgresql", "plugin:redis"]
}
```
