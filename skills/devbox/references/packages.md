# Package Management

## Finding a package

```bash
devbox search <name>          # searches the Nix registry
devbox search nodejs
devbox info nodejs@20         # details on a specific version
```

[nixhub.io](https://www.nixhub.io) is the web UI for the same registry — useful for browsing available versions.

## Adding packages

```bash
devbox add go@1.22                          # pinned version
devbox add nodejs@latest                    # track latest
devbox add postgresql --platform=x86_64-linux,aarch64-linux   # allowlist
devbox add ffmpeg --exclude-platform=aarch64-darwin            # denylist
devbox add bun --outputs=bin,lib            # select Nix outputs
devbox add openssl@1.1 --allow-insecure openssl-1.1.1w         # unblock CVE-flagged pkg
devbox add go --disable-plugin              # skip built-in plugin
```

Always pin a version in committed configs. `@latest` is fine for one-offs and global tools but makes project builds non-reproducible until `devbox update` runs.

## Removing packages

```bash
devbox rm <name>              # single package
devbox rm go nodejs           # multiple
```

This updates `devbox.json` and `devbox.lock`. Re-run `devbox install` on other machines to remove the store paths.

## Updating packages

```bash
devbox update                       # update all packages
devbox update go                    # update one
devbox update --no-install          # update lockfile only, skip download
devbox update --all-projects        # recurse into every devbox project in cwd
devbox update --sync-lock           # reconcile lockfiles across multiple projects
```

`devbox update` also migrates legacy non-versioned packages (e.g. `"go"`) to `@latest` form, resolving to the current version in the process.

## Installing (materializing the lockfile)

```bash
devbox install                   # install everything in devbox.lock, no shell
devbox install --tidy-lockfile   # repair missing store paths
```

Use `devbox install` in CI and Dockerfiles. It's the non-interactive equivalent of entering a shell.

## Versioning semantics

- `go@1.22` — pins to the highest `1.22.x` available.
- `go@1.22.5` — pins exactly.
- `go@latest` — tracks whatever is latest when `devbox update` runs.
- Unversioned (`go`) — legacy form; still works but migrates to `@latest` on next update.

Versions are resolved against Nixhub, which aggregates snapshots of nixpkgs over time, so older versions remain reachable long after they leave `nixpkgs-unstable`.

## Platform-specific packages

Two mutually exclusive keys control where a package installs:

- `platforms`: allowlist.
- `excluded_platforms`: denylist.

```json
"packages": {
  "podman":      { "version": "latest", "platforms": ["x86_64-linux", "aarch64-linux"] },
  "docker-mac":  { "version": "latest", "platforms": ["aarch64-darwin", "x86_64-darwin"] }
}
```

Useful when different OSes need different tools to achieve the same capability.

## `runx` packages

Devbox supports `runx:<github-owner>/<repo>[@version]` to pull prebuilt release binaries from GitHub releases. Useful for tools not in nixpkgs (e.g. `runx:golangci/golangci-lint@latest`). They install on demand and run without a full Nix build.

## Insecure packages

If Nix marks a package as insecure (e.g. an old OpenSSL), add it with:

```bash
devbox add openssl@1.1 --allow-insecure openssl-1.1.1w
```

The allowlist goes into `devbox.json` as `allow_insecure`. Pin the exact attribute name shown in the error message.
