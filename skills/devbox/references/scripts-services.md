# Scripts, Services, and Shell Hooks

## Scripts via `devbox run`

Scripts live under `shell.scripts` in `devbox.json`:

```json
"shell": {
  "scripts": {
    "test":  "go test ./...",
    "build": ["go mod tidy", "go build -o bin/app ./cmd/app"],
    "dev":   "air"
  }
}
```

Run them with:

```bash
devbox run test
devbox run -- build --verbose     # args after -- are forwarded
devbox run --list                 # list scripts
devbox run --all-projects test    # recurse
devbox run --pure test            # isolated env (drops host vars)
devbox run --env KEY=val test     # ad-hoc env vars
devbox run --env-file ./.env test
```

If the name isn't in `scripts`, `devbox run` interprets it as an arbitrary command in the devbox environment.

### Gotchas

- **Pass flags to the underlying command with `--`.** Otherwise devbox parses them itself.
- **Arrays stop on first failure** (non-zero exit) — same as `set -e`. If you need to continue past failures, combine with `||` inside the string.
- **`init_hook` runs before every script** — including inside CI. Keep it lightweight.

## Services

`devbox services` orchestrates long-running processes (databases, workers) via [process-compose](https://github.com/F1bonacc1/process-compose).

```bash
devbox services ls                 # list services
devbox services start postgresql   # start one
devbox services start              # start all
devbox services up                 # start with process-compose UI
devbox services up -b              # background
devbox services restart redis
devbox services stop               # stop all
devbox services attach             # attach to the running process-compose
devbox services pcport             # print process-compose port
```

Services come from two sources:

1. **Plugins** — e.g. `"include": ["plugin:postgresql"]` registers a `postgresql` service with sensible defaults (data dir under `.devbox/virtenv/postgresql/`, port config, etc.).
2. **Custom** — drop a `process-compose.yaml` in the project root to define your own services.

Override plugin env vars with `--env`, `--env-file`, or the project's `env` block.

### Typical dev loop

```bash
devbox services up -b             # boot Postgres + Redis in background
devbox run dev                    # run the app against them
devbox services stop              # tear down when done
```

## Shell init hook

`shell.init_hook` runs every time the environment starts — both `devbox shell` and every `devbox run`.

```json
"shell": {
  "init_hook": [
    "test -z $FISH_VERSION && unset GOROOT GOTOOLCHAIN",
    "export PROJECT_ROOT=$PWD"
  ]
}
```

Good uses:

- Unset host env vars that would poison the build (e.g. stale `GOROOT`).
- Export computed values that can't live in the static `env` block.
- Create local directories or install tool binaries that are cheap to check.

Avoid:

- Expensive work — it runs on every invocation.
- Long network calls — they'll slow every `devbox run` in CI.
- Interactive prompts — breaks non-interactive use.

## `devbox shellenv`

Prints shell commands that load the devbox environment into the *current* shell without spawning a subshell:

```bash
eval "$(devbox shellenv)"
```

Useful for:

- Scripts that need devbox packages without `devbox run` wrapping.
- Shell configs (`.zshrc`) that want a project's devbox env loaded at startup.
- CI when you want the devbox env active for several steps.

The nicer wrapper is `devbox generate direnv`, which produces a `.envrc` so direnv loads the env automatically on `cd`.
