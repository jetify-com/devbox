---
name: use-devbox
description: Run commands inside a devbox environment when a directory provides one. If a `devbox.json` exists in the current directory (or an ancestor/subdir root), devbox can supply tools and scripts the bare shell lacks — e.g. psql, python, gcloud, node. If a binary is provided by devbox, prefer it over the system binary.
---

# Use devbox

`devbox` applies to directories that have a `devbox.json` (at the directory root or a subdir root). Outside such directories, devbox does nothing useful.

[devbox](https://github.com/jetify-com/devbox) creates isolated, reproducible development environments from a `devbox.json` file. The packages it lists come from the Nix package registry and are made available *inside* the devbox environment — not the bare shell.

## When to reach for it

- The directory has a `devbox.json`.
- Before concluding a tool is unavailable, check for a `devbox.json` — the project may install that tool through devbox.
- The user explicitly mentions devbox.

## Check it's installed

devbox is not always installed. Check first:

```bash
command -v devbox
```

If that prints nothing, install it (non-interactive):

```bash
curl -fsSL https://get.jetify.com/devbox | bash
```

devbox installs to `~/.local/bin` (or `/usr/local/bin`). If `devbox` still isn't found after installing, ensure that directory is on `PATH`. devbox itself requires the Nix package manager; the install script sets it up if missing.

## How to detect

Look for a `devbox.json` in the working directory or an ancestor:

```bash
ls devbox.json 2>/dev/null || find . -maxdepth 2 -name devbox.json 2>/dev/null
```

If one exists, the packages it lists are available *inside* the devbox environment (not the bare shell).

## How to run

Run any command inside the devbox environment with:

```bash
devbox run [cmd]
```

Examples:

```bash
devbox run psql "$MASTER_DB_URL" -c 'SHOW wal_level;'
devbox run python script.py
devbox run gcloud sql instances describe <instance>
```

`devbox run` is the right choice for agents and scripts: it loads the environment, runs one command non-interactively, and exits. The first run in a fresh environment may be slow while packages download/build; later runs are cached and fast.

If a directory other than the cwd holds the `devbox.json`, run from that directory (e.g. `devbox run --config path/to/devbox.json [cmd]` or `cd` into it first).

## Project scripts

`devbox.json` can define named scripts under a `"shell": { "scripts": { ... } }` block. Run them by name — `devbox run <script-name>` — instead of retyping the underlying command. For example, this repo defines `test`, `lint`, `fmt`, `build`, and `tidy`, so `devbox run test` runs the project's test suite inside the environment.

To see what's defined, run `devbox run` with no script name (it lists available scripts) or inspect `devbox.json`.

## Choosing where a binary comes from

When you need a binary, decide its source in this order:

1. **Provided by devbox for the current project** — if the project's `devbox.json` supplies it, prefer the devbox version over any system install. Run it via `devbox run [cmd]`.
2. **Provided by the system but not devbox** — use the system binary.
3. **Not provided by devbox or the system** — check whether devbox can supply it:

   ```bash
   devbox search [pkg]
   ```

   If it's available, **ask the user** whether they want to install it into the current project before doing so. Only on their confirmation:

   ```bash
   devbox add [pkg]
   ```

   Don't add packages to a project's `devbox.json` without the user's go-ahead. (`devbox rm [pkg]` removes one.)

## Services

Some projects define long-running services (databases, queues, etc.) in their devbox config:

```bash
devbox services ls          # list defined services
devbox services up           # start services (add -b to run in background)
devbox services stop         # stop them
```

## Global packages

`devbox global` manages a machine-wide environment that's independent of any project (`devbox global add/rm/list`). Prefer per-project packages unless the user explicitly wants something available everywhere.

## Discovering more

- `devbox --help` — full command list (shell, run, add, services, etc.).
- `devbox run` without args, or inspect `devbox.json`, to see project-defined scripts.
- `devbox shell` opens an interactive subshell with the environment loaded (prefer `devbox run` for one-off, non-interactive commands).
- To see what binaries devbox is adding, run `devbox install` and then inspect the `.devbox/nix/profile/default/bin/` directory.

## Key facts

- devbox is not always installed — check with `command -v devbox` and install via `curl -fsSL https://get.jetify.com/devbox | bash` if missing.
- Only directories with a `devbox.json` use it.
- `devbox.json` may live in the repo root **or** a subdir root — check both.
- Packages/scripts defined in `devbox.json` are only on PATH inside `devbox run`/`devbox shell`, not the plain shell.
- Prefer `devbox run` over `devbox shell` for non-interactive, one-off commands.
- The first run after adding packages or in a fresh checkout can be slow (downloading/building); subsequent runs are cached.
