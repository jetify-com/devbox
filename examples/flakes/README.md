# Flakes

Examples that show how to add custom flakes to your devbox project.

## Local flakes (usually committed to your project)

In devbox.json use "path:/path/to/flake#output" as the package name.

```json
{
  "packages": [
    "path:my-php-flake#php",
    "path:my-php-flake#hello"
  ],
  "shell": {
    "init_hook": null
  },
  "nixpkgs": {
    "commit": "f80ac848e3d6f0c12c52758c0f25c10c97ca3b62"
  }
}
```

This installs the "php" and "hello" outputs from the flake at `my-php-flake`. These outputs can also be part of packages or legacyPackages.

## Remote flakes

Use `github:<org>/<repo>/<ref>#<output>` as the package name to install from a Github repo.

```json
{
  "packages": [
    "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello",
    "github:nix-community/fenix#stable.toolchain",
    "github:F1bonacc1/process-compose"
  ],
  "shell": {
    "init_hook": null
  },
  "nixpkgs": {
    "commit": "f80ac848e3d6f0c12c52758c0f25c10c97ca3b62"
  }
}
```

This installs the `hello` package from the 5233fd... commit of Nixpkgs, the `stable.toolchain` output from the `fenix` package in the `nix-community/fenix` repo, and the `default` output from the `F1bonacc1/process-compose` repo.
