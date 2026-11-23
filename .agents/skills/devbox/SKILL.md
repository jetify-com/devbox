---
name: devbox
description: Use whenever a project contains a devbox.json, or the user mentions devbox. Run builds, tests, and tools through `devbox run` so they use the project's pinned packages; add tools with `devbox add` instead of brew/apt/global installs; and fix missing commands, native library or linker errors, and service setup (postgres, redis, mysql, ...) the devbox way.
metadata:
  managed-by: devbox
  devbox-version: "0.0.0-dev"
---

# Devbox

Devbox builds a reproducible dev environment from `devbox.json` (packages come
from Nix). Packages are only on `PATH` inside the environment, not the bare shell.

## Rule: if `devbox.json` exists, use devbox

Look in the working directory and its ancestors. If one exists:

- Run commands through devbox: `devbox run -- <cmd> [args]`. Do not assume a
  system `node`, `python`, `go`, `psql`, etc. is the right one.
- Prefer project scripts: `devbox run --list`, then `devbox run <script>`.
- Scripts and commands run from the **project root**, not the cwd. For a
  subdirectory: `devbox run -- sh -c 'cd sub && <cmd>'`.
- Never install project dependencies with `brew`, `apt`, `sudo`, `nix-env`,
  `npm -g`, or system `pip`. Add them to the project with `devbox add`.
- Never edit `.devbox/` (generated). Commit `devbox.json` and `devbox.lock`.

If `devbox` is missing: `curl -fsSL https://raw.githubusercontent.com/jetify-com/devbox/main/install.sh | bash`.
The first run in a fresh checkout can take minutes while packages download.

## Packages

```bash
devbox search <name>            # find packages and available versions
devbox add <pkg>@<version>      # e.g. nodejs@20, python@3.12, go@latest
devbox rm <pkg>
devbox list                     # what this project has
devbox info <pkg>               # versions, plugin notes, services
devbox update [<pkg>]           # bump to the newest match; rewrites devbox.lock
```

- Pin a major/minor (`nodejs@20`) rather than `latest` when the project needs
  stability. The exact version is locked in `devbox.lock`.
- Ask before adding packages unless the user asked for the change.
- Add `-p <platform>` / `-e <platform>` (e.g. `x86_64-linux`,
  `aarch64-darwin`) for packages that only exist on some platforms.
- Use `devbox global add` only for personal tools the user wants everywhere.

## Plugins and services

Many packages ship a built-in plugin that activates automatically on `add`
(postgresql, mysql, mariadb, redis, valkey, nginx, caddy, apache, php, python,
nodejs, ruby, rust, haskell, ...). Plugins set env vars, create config under
`devbox.d/`, and define services. Run `devbox info <pkg>` to read its notes.

- Prefer the plugin over hand-written setup: e.g. `devbox add postgresql`, then
  `devbox services up -b`, instead of installing and configuring Postgres yourself.
- Customize via the files the plugin writes to `devbox.d/<pkg>/` (committed) or
  via `env` in `devbox.json`. Service data lives in `.devbox/virtenv/`.
- Disable a plugin you don't want with `devbox add <pkg> --disable-plugin`
  (sets `"disable_plugin": true`).
- Share setup across projects by writing a plugin and listing it in
  `"include"`: `"path:./plugins/my-plugin"` or `"github:org/repo?dir=plugin"`.

```bash
devbox services ls
devbox services up -b           # start all in the background (needed for agents)
devbox services stop
```

## devbox.json essentials

```jsonc
{
  "packages": { "python": "3.12", "postgresql": "latest" },
  "env": { "DATABASE_URL": "postgresql://localhost/app" },  // supports $VAR and ${VAR:-default}
  "env_from": ".env",
  "shell": {
    "init_hook": ["echo runs before every shell, script, and service"],
    "scripts": { "test": "pytest", "dev": ["npm install", "npm run dev"] }
  }
}
```

Keep `init_hook` fast and idempotent; it runs on every `devbox run`. Put
one-off setup in a script instead.

## Language notes

- **Python**: the plugin creates a venv at `$VENV_DIR` (`.venv`). Install with
  `devbox run -- pip install ...` (or `uv`), never the system pip.
- **Node**: add `nodejs@<v>`; pnpm/yarn come via Corepack (plugin). Use
  `nodejs-slim` with `--disable-plugin` if you only need `node`.
- **Go/Rust**: add the toolchain; module/crate deps are managed as usual.

## Native libraries and compatibility

The usual cause is a native library outside the Nix environment.

1. **Build-time headers/libs** (`fatal error: openssl/ssl.h`, `cannot find -lz`,
   `pkg-config` failures): add the library *and* `pkg-config`, e.g.
   `devbox add openssl zlib libffi pkg-config`. If headers are still missing,
   select the dev output: `devbox add openssl --outputs=out,dev`.
2. **Run-time loading** of prebuilt binaries (pip wheels, npm native addons,
   downloaded tools) failing with `libstdc++.so.6: cannot open shared object`,
   `GLIBC_2.xx not found`, or `libz.so.1` on Linux:
   - Add the library (`stdenv.cc.cc.lib` provides libstdc++, `zlib`, ...) and
     expose it: `"env": {"LD_LIBRARY_PATH": "$DEVBOX_PACKAGES_DIR/lib:$LD_LIBRARY_PATH"}`.
   - For a devbox package with a mismatched glibc, set `"patch": "always"`
     on it (`devbox add <pkg> --patch always`). Python is patched by default.
   - Prefer building from source or a Nix-packaged version over prebuilt binaries.
3. **macOS** linker/clang errors (cgo, Rust `-liconv`, frameworks): add the
   missing library (`libiconv`, ...) or, for Go, try `CGO_ENABLED=0` if cgo
   isn't needed.
4. **Insecure** packages (e.g. EOL versions): `devbox add <pkg> --allow-insecure <pkg>`.
   Unfree packages are allowed automatically.
5. **Wrong binary picked up**: check with `devbox run -- which <cmd>`; devbox
   packages live in `$DEVBOX_PACKAGES_DIR/bin`. Inspect the full env with
   `devbox shellenv`.

When stuck, rerun with `--debug`, run `devbox install` to rebuild, and check
`devbox info <pkg>`. Docs: https://www.jetify.com/docs/devbox/
