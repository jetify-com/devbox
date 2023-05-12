---
title: Selecting a Specific Package Version
---

This document explains how to use `devbox search` and `devbox add` to install a specific package version in your Devbox project. It also explains how to pin a particular major or minor version for the package in your project.

## The Nixpkgs Repository and the Devbox Search Index

Devbox installs packages using the [Nix Package Manager](nixos.org). Nix maintains over 80,000 build definitions in a Github repo at [NixOS/nixpkgs](https://github.com/NixOS/nixpkgs). Maintainers add new packages and remove outdated packages by committing changes to this repo.

Because the repository changes frequently, and new releases of Nixpkgs infrequently keep older packages, installing older package versions with Nix can take effort. Devbox simplifies this by maintaining a search index that maps package names and version numbers to their latest available commit in the Nixpkgs repository. Devbox users can select packages by providing the package name and version without looking up a nixpkg commit.

## Pinning a Package Version

To pin a specific version of a package, you can add a `@` followed by the version number at the end of the package name. For example, to pin the `go` package to version `1.19`, you can run `devbox add go@1.19` or add `go@1.19` to the packages list in your `devbox.json`:

```json
"packages": [
	"go@1.19"
]
```

For packages that use semver, you can pin a range of versions for your project. For example, if you pin `python@3`, it will install the latest minor and patch version of `python >=3.0.0`. You can update to the newest package version that matches your criteria by running `devbox update`.

You can look up the available versions of a package by running `devbox search <package_name>`.

### Updating your packages

If you want to update your packages, you can run `devbox update`. This command will update all your pinned packages to the newest compatible version in the Devbox index.

### Using the Latest Version of a Package

To ensure you use the latest available package, you can run `devbox add <pkg>` without including a version string or adding `package_name@latest` to the package list in your devbox.json. For example, to use the latest version of `ripgrep,` run `devbox add ripgrep` or add `ripgrep@latest` to your devbox.json.

Whenever you run `devbox update`, your package will be updated to the latest version available in our index.

## Manually Pinning a Nixpkg Commit for a Single Package

If you want to use a different commit for a single package, you can use a Flake reference to use an older revision of Nixpkg for just that package. The example below shows how to install the `hello` package from a specific Nixpkg commit:

```json
}
	"packages" : [
"github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello"
	]
}
```
Using multiple nixpkg commits may install duplicate packages and cause Nix Store bloat, so use this option sparingly.

## Pinning the Default Nixpkg commit in your Devbox.json

::: note
Pinning the nixpkgs commit is deprecated as of version 0.5.0 and will eventually be removed. We recommend using the `@` syntax to pin packages.
:::

Devbox stores a default Nixpkg commit in your project's `devbox.json`, under the `nixpkgs.commit`. If you do not provide one yourself, Devbox will automatically add a default commit when you run a command like `devbox add`, `devbox shell`, or `devbox run`:

```json
"nixpkgs": {
    "commit": "89f196fe781c53cb50fef61d3063fa5e8d61b6e5"
}
```
This hash ensures that Devbox will install the same packages whenever you start a shell. By checking this into source control, you can ensure that other developers who run your project will get the same packages.
