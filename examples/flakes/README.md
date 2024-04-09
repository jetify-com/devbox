# Flakes

Examples that show how to add custom flakes to your Devbox project. These examples require [Devbox 0.4.7](https://www.jetify.com/blog/devbox-0-4-7/) or later.

For more details, you can also consult our Docs page on [using flakes](https://www.jetify.com/devbox/docs/guides/using_flakes/)

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

This installs the `hello` package from the 5233fd... commit of Nixpkgs, and the `default` output from the `F1bonacc1/process-compose` repo.
