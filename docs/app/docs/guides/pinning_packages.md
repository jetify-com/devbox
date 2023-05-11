---
title: Selecting a Specific Package Version
---

This doc will explain how to select and pin specific package versions in Devbox by setting a Nixpkg commit in your devbox.json

## Context on Nixpkgs

The Nix Package Manager, which Devbox uses to install your shell packages, stores its package definitions in a Github Repository at [NixOS/nixpkgs](https://github.com/NixOS/nixpkgs). This repository contains instructions for building over 80,000 different packages. Maintainers add new packages or remove deprecated packages by committing to the repo.

## Pinning a Package Version

To pin a specific version of a package, you can add a `@` followed by the version number to the end of the package name. For example, to pin the `go` package to version `1.19`, you can run `devbox add go@1.19` or add `go@1.19` to the packages list in your `devbox.json`:

```json
"packages": [
	"go@1.19"
]
```

Pinned packages that follow semver will install the latest version of the package with the same major version. For example, if you pin `python@3`, it will install the latest version of `python` with major version `3`.

You can look up the available versions of a package by running `devbox search <package_name>`.

### Updating your packages

If you want to update your packages to the latest version, you can run `devbox update`. This will update all of your pinned packages to the latest version.

## Manually Pinning a Nixpkg Commit for a Single Package

If you want to use a different commit for a single package, you can use a Flake reference to use an older revision of Nixpkg for just that package. The example below shows how to install the `hello` package from a specific Nixpkg commit:

```json
}
	"packages" : [
"github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello"
	]
}
```
Note that using a different nixpkg commit may install duplicate packages and cause Nix Store bloat, so use this option sparingly.

## Pinning the Default Nixpkg commit in your Devbox.json

Devbox stores the Nixpkg commit in your project's `devbox.json`, under the `nixpkgs.commit`. If you do not specify one in your config, Devbox will automatically add a default commit hash when you run a command like `devbox add`, `devbox shell`, or `devbox run`:

```json
"nixpkgs": {
    "commit": "89f196fe781c53cb50fef61d3063fa5e8d61b6e5"
}
```
This hash ensures that Devbox will install the same packages whenever you start a shell. By checking this into source control, you can also ensure that any other developers who run your project will get the same packages.

### Using the Latest Version of Nixpkgs

To use the latest available packages in Nix, you can replace the commit in `devbox.json` with the latest `nixpkgs-unstable` hash from [https://status.nixos.org](https://status.nixos.org).
