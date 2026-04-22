# Global Packages, Templates, and Generators

## `devbox global` — machine-wide tools

`devbox global` manages a separate config (kept at `$(devbox global path)`) that's always on your PATH, independent of any project. Mirror of the per-project commands:

```bash
devbox global add jq ripgrep fzf        # add tools
devbox global list                      # show installed
devbox global rm jq
devbox global install                   # reinstall from config
devbox global update                    # update all
devbox global path                      # print config directory
```

To make it live in your shell, add one of:

```bash
eval "$(devbox global shellenv)"        # in ~/.zshrc or ~/.bashrc
```

### Sharing a global config

```bash
devbox global push                      # push to Jetify Cloud
devbox global push git@github.com:me/dotfiles-devbox.git   # or your own git repo
devbox global pull <url-or-file>        # pull on another machine
```

Useful for keeping the same CLI toolkit across laptops.

**Global and project configs never mix.** A project's `devbox.json` doesn't see global packages unless you explicitly run `eval "$(devbox global shellenv)"` in your shell first.

## `devbox create` — templates

```bash
devbox create --show-all                        # list templates
devbox create my-proj --template go             # scaffold into ./my-proj
devbox create . --template python               # scaffold into cwd
```

Templates are starter `devbox.json` + supporting files for common stacks (go, python, node, rust, etc.). Use when starting fresh.

## `devbox generate` — supporting files

```bash
devbox generate readme          # regenerate the project README from devbox.json
devbox generate dockerfile      # Dockerfile that replicates the shell
devbox generate devcontainer    # .devcontainer/ for VS Code
devbox generate direnv          # .envrc for direnv auto-loading
devbox generate alias           # shell aliases for scripts
```

### `generate dockerfile`

Produces a `Dockerfile` that installs Nix, runs `devbox install`, and sets the entrypoint. The resulting image reproduces the dev shell. Pair with `.dockerignore` to avoid shipping `.devbox/` build artifacts.

### `generate devcontainer`

Writes `.devcontainer/devcontainer.json` and `.devcontainer/Dockerfile`. VS Code's Remote-Containers will pick it up and build an identical environment inside Docker.

### `generate direnv`

Writes a `.envrc` at the project root that runs `use devbox`. After `direnv allow`, the devbox environment activates automatically when you `cd` into the directory and deactivates when you leave. The recommended daily-use setup.

### `generate readme`

Regenerates the auto-generated README section (the `<!-- gen-readme start -->` block) from the current `devbox.json` — scripts, packages, env, init hook. Run after changing `devbox.json` to keep docs fresh.

## Secrets

`devbox secrets` integrates with Jetify Cloud for encrypted secret storage per environment:

```bash
devbox secrets init                     # link project to a Jetify secrets store
devbox secrets set API_KEY=sk-...       # store a secret
devbox secrets list --environment prod  # list for an environment
devbox secrets download .env.local      # materialize to a file
devbox secrets upload .env              # bulk import from a .env file
devbox secrets remove API_KEY
```

Environments (`dev` / `prod` / `preview`) scope secrets separately. Pass `--environment <name>` on `run`, `shell`, and `secrets` commands.

## Auth

```bash
devbox auth login         # log in to Jetify Cloud (needed for secrets, global push)
devbox auth logout
devbox auth whoami
```
