---
name: devbox
description: Devbox expert guidance. Use when creating isolated development environments, managing project dependencies with Nix packages, authoring devbox.json, writing scripts or services, or generating Dockerfiles and devcontainers from a devbox project.
metadata:
  docs:
    - "https://www.jetify.com/devbox/docs"
    - "https://github.com/jetify-com/devbox"
  pathPatterns:
    - 'devbox.json'
    - 'devbox.lock'
    - '.devbox/**'
  bashPatterns:
    - '^\s*devbox(?:\s|$)'
---

# Devbox Skill

[Devbox](https://github.com/jetify-com/devbox) creates isolated, reproducible development environments powered by Nix. You declare packages and scripts in a `devbox.json`; Devbox materializes a shell where everyone on the team gets the exact same tool versions without polluting the host machine. Use `devbox <command> -h` for full flag details on any command.

## Quick Start

```bash
curl -fsSL https://get.jetify.com/devbox | bash   # install devbox
devbox init                                        # create devbox.json in cwd
devbox add go@1.22 nodejs@20                       # add packages (name@version)
devbox shell                                       # enter the isolated shell
devbox run <script>                                # run a script from devbox.json
```

Packages resolve against the Nix package registry. Browse versions at [nixhub.io](https://www.nixhub.io).

## The `devbox.json` File

The project's source of truth. Minimum viable file:

```json
{
  "packages": ["go@1.22", "nodejs@20"],
  "shell": {
    "init_hook": ["echo 'Welcome'"],
    "scripts": {
      "test": "go test ./...",
      "build": ["go build ./...", "echo done"]
    }
  },
  "env": {
    "GOENV": "off",
    "PATH": "$PWD/bin:$PATH"
  }
}
```

- `packages`: array of `name@version` strings, or an object with per-package `platforms` / `excluded_platforms` / `outputs` / `patch` settings.
- `shell.init_hook`: commands run every time the environment starts (both `devbox shell` and `devbox run`). Keep it fast — it runs often.
- `shell.scripts`: named commands invoked with `devbox run <name>`. A value can be a string or an array of strings executed in order.
- `env`: extra env vars. Only `$PATH` and `$PWD` are expanded; no other variable expansion or command substitution.
- `include`: plugin references (e.g. `"plugin:nginx"`, `"path:./mydir"`, `"github:owner/repo"`).

See `references/devbox-json.md` for the full schema.

## Decision Tree

Use this to route to the correct sub-topic:

- **Add or remove a package** → `devbox add <pkg>@<ver>` / `devbox rm <pkg>`. See `references/packages.md`.
- **Enter an isolated shell** → `devbox shell`. Add `--pure` for a shell that inherits almost nothing from the host.
- **Run a task** → `devbox run <script>`. Pass arguments after `--` (e.g. `devbox run -- cowsay -d hi`). See `references/scripts-services.md`.
- **Install all packages in CI** → `devbox install` (no shell). Pair with `actions/cache` keyed on `devbox.lock`.
- **Update packages** → `devbox update` (all) or `devbox update <pkg>` (one). Use `--no-install` to update the lockfile only.
- **Find a package** → `devbox search <pkg>`.
- **Inspect installed packages** → `devbox list` / `devbox info <pkg>`.
- **Long-running services (dbs, workers)** → `devbox services start|stop|ls|attach`. See `references/scripts-services.md`.
- **Global tools (available everywhere)** → `devbox global add|rm|list`. See `references/global-and-templates.md`.
- **Bootstrap from a template** → `devbox create --template <name>` (use `--show-all` to list templates).
- **Generate Dockerfile / devcontainer / direnv** → `devbox generate dockerfile|devcontainer|direnv|alias|readme`.
- **Load env into the host shell** → `eval "$(devbox shellenv)"`, or `devbox generate direnv` for automatic loading.
- **Secrets (Jetify Cloud)** → `devbox secrets init|set|list|download|upload`.
- **Multiple envs (dev/prod/preview)** → `--environment <name>` flag.
- **Multi-project repos** → `--all-projects` on `run` and `update`; `--sync-lock` on `update`.

## Critical: Lockfile Hygiene

`devbox.lock` pins exact Nix store paths for every package. **Always commit it.** Without it, teammates and CI will get different versions.

- `devbox install --tidy-lockfile` repairs missing store paths.
- `devbox update --sync-lock` reconciles lockfiles across multiple projects in one repo.
- Do not hand-edit `devbox.lock`.

## Integrating with the Host Shell

`devbox shell` spawns a subshell, which is noisy for daily work. Two better options:

1. **direnv**: `devbox generate direnv` writes a `.envrc` so entering the directory auto-loads the environment. Requires `direnv` installed on the host.
2. **shellenv**: `eval "$(devbox shellenv)"` inline-loads the environment into the current shell. Useful in CI or one-off scripts.

## CI/CD

Typical GitHub Actions pattern:

```yaml
- uses: jetify-com/devbox-install-action@v0.13.0
  with:
    enable-cache: 'true'
- run: devbox run test
```

Without the action, use `devbox install` + cache `~/.nix-*` and `.devbox/` keyed on `devbox.lock`. Avoid `devbox shell` in CI — use `devbox run <cmd>` (non-interactive) or `eval "$(devbox shellenv)"` instead.

## Anti-Patterns

- **Running devbox without a lockfile committed.** Defeats reproducibility. Commit `devbox.lock` alongside `devbox.json`.
- **Using unversioned packages in production configs** (e.g. `"go"` instead of `"go@1.22"`). The resolver will pick whatever `@latest` resolves to the day you first install it; `devbox update` is the only way to bump it. Pin versions.
- **Shell substitution in `env` values.** Only `$PATH` / `$PWD` expand. Put dynamic logic in `init_hook` instead.
- **Heavy work in `init_hook`.** It runs on *every* `devbox run`, not just shell entry. Keep it to env mutations; move builds to scripts.
- **`devbox shell` inside scripts or CI.** Blocks on an interactive prompt. Use `devbox run` or `devbox shellenv` instead.
- **Editing `devbox.lock` by hand.** Let `devbox update`, `devbox install --tidy-lockfile`, or `devbox add` regenerate it.
- **Forgetting `--` before flags to the target command.** `devbox run cowsay -d hi` tries to parse `-d` as a devbox flag. Use `devbox run -- cowsay -d hi`.
- **Mixing `devbox` and `devbox global`.** `global` manages a separate config at `devbox global path` — project installs never leak into it, and vice versa.

## Common Recipes

- **New Go project**: `devbox init && devbox add go@latest && devbox shell`
- **Dockerize**: `devbox generate dockerfile` produces a `Dockerfile` that replicates the shell.
- **Reproduce in a devcontainer**: `devbox generate devcontainer` writes `.devcontainer/` for VS Code.
- **Share a global toolset**: `devbox global push` (to Jetify Cloud or a git repo); teammates run `devbox global pull`.
- **Run Postgres for local dev**: `devbox add postgresql` then `devbox services up postgresql`.
