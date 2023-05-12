---
title: Selecting a Specific Package Version
---

This document explains how to use `devbox search` and `devbox add` to install a specific package version in your Devbox project. It also explains how to pin a particular major or minor version for the package in your project.

## The Nixpkgs Repository and the Devbox Search Index

Devbox installs packages using the [Nix Package Manager](https://nixos.org). Nix maintains over 80,000 build definitions in a Github repo at [NixOS/nixpkgs](https://github.com/NixOS/nixpkgs). Maintainers add new packages and remove outdated packages by committing changes to this repo.

Because the repository changes frequently, and new releases of Nixpkgs infrequently keep older packages, installing older package versions with Nix can take effort. Devbox simplifies this by maintaining a search index that maps package names and version numbers to their latest available commit in the Nixpkgs repository. Devbox users can select packages by providing the package name and version without looking up a nixpkg commit.

## Pinning a Package Version

### Searching for Available Packages

You can look up the available versions of a package by running `devbox search <package_name>`. For example, to see the available versions of `python`, you can run `devbox search python`:

```bash
$ devbox search python

Found 2770+ results for "python":

* python (3.12.0a7, 3.12.0a6, 3.12.0a5, 3.12.0a3, 3.11.3, 3.11.2, 3.11.1, 3.11.0, 3.11.0rc1, 3.11.0b3, 3.11.0a7, 3.11.0a4, 3.11.0a2, 3.10.4, 3.10.2, 3.10.0, 3.10.0rc1, 3.10.0a5, 3.10.0a3, 3.10.0a1, 3.9.16, 3.9.14, 3.9.13, 3.9.4, 3.9.2, 3.9.1, 3.9.0, 3.9.0b5, 3.9.0a4, 3.8.16, 3.8.15, 3.8.13, 3.8.12, 3.8.11, 3.8.8, 3.8.6, 3.8.5, 3.8.3, 3.7.16, 3.7.15, 3.7.13, 3.7.12, 3.7.11, 3.7.10, 3.7.9, 3.7.8, 3.7.7, 3.6.14, 3.6.13, 3.6.12, 3.6.11, 3.6.10, 3.5.9, 2.7.18.6, 2.7.18.5, 2.7.18)
...
```

### Adding a Specific Version to Devbox

To add a specific version of a package with `<package_name>@<version>`. For example, to pin the `python` package to version `3.11.1`, you can run `devbox add python@3.11.1` or add `python@3.11.1` to the packages list in your `devbox.json`:

```json
"packages": [
	"python@3.11.1"
]
```

For packages that use semver, you can pin a range of versions for your project. For example, if you pin `python@3`, it will install the latest minor and patch version of `python >=3.0.0`. You can update to the newest package version that matches your criteria by running `devbox update`.

When you run a command that installs your packages (like `devbox shell` or `devbox install`), Devbox will generate a `Devbox.lock` file that contains the exact version and commit hash for your packages. You should check this file into source control to ensure that other developers will get the same environment.

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
