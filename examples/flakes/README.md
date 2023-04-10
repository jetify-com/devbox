# Flakes

Examples that show how to add custom flakes to your devbox project.

# Local flakes (usually committed to your project)

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

# Remote flakes

TODO
