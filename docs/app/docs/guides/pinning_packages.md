---
title: Installing a Specific Package Version
---

This document explains how to use `devbox search` and `devbox add` to install a specific package version in your Devbox project. It also explains how to pin a particular major or minor version for the package in your project.

## The Nixpkgs Repository and the Devbox Search Index

Devbox installs packages using the [Nix Package Manager](https://nixos.org). Nix maintains over 80,000 build definitions in a Github repo at [NixOS/nixpkgs](https://github.com/NixOS/nixpkgs). Maintainers add new packages and remove outdated packages by committing changes to this repo.

Because the repository changes frequently, and new releases of Nixpkgs infrequently keep older packages, installing older package versions with Nix can take effort. Devbox simplifies this by maintaining a search index that maps package names and version numbers to their latest available commit in the Nixpkgs repository. Devbox users can select packages by providing the package name and version without looking up a Nixpkg commit.

## Pinning a Package Version

### Searching for Available Packages

You can look up the available versions of a package by running `devbox search <package_name>`. For example, to see the available versions of `nodejs`, you can run `devbox search nodejs`:

```bash
$ devbox search nodejs

Found 2+ results for "nodejs":

* nodejs  (20.5.1, 20.5.0, 20.4.0, 20.3.1, 20.3.0, 20.2.0, 20.1.0, 20.0.0, 19.9.0, 19.8.1)
* nodejs-slim  (20.5.1, 20.5.0, 20.4.0, 20.3.1, 20.3.0, 20.2.0, 20.1.0, 20.0.0, 19.9.0, 19.8.1)

Warning: Showing top 10 results and truncated versions. Use --show-all to show all.
```

### Adding a Specific Version to Devbox

To add a specific version of a package with `<package_name>@<version>`. For example, to pin the `nodejs` package to version `19.9.0`, you can run `devbox add nodejs@19.9.0` or add `nodejs@19.9.0` to the packages list in your `devbox.json`:

```json
"packages": [
	"nodejs@19.9.0"
]
```

For packages that use semver, you can pin a range of versions for your project. For example, if you pin `nodejs@19`, it will install the latest minor and patch version of `nodejs >=19.0.0`. You can update to the newest package version that matches your criteria by running `devbox update`.

When you run a command that installs your packages (like `devbox shell` or `devbox install`), Devbox will generate a `devbox.lock` file that contains the exact version and commit hash for your packages. You should check this file into source control to ensure that other developers will get the same environment.

### Updating your packages

If you want to update your packages, you can run `devbox update`. This command will update all your pinned packages to the newest compatible version in the Devbox index.

### Using the Latest Version of a Package
If you do not include a version string, Devbox will default to using the latest available version of the package in our Nixpkg index. This is the same as adding `<pkg>@<latest>` to your devbox.json.

For example, to use the latest version of `ripgrep,` run `devbox add ripgrep`, `devbox add ripgrep@latest`, or add `ripgrep@latest` to your devbox.json package list.

Whenever you run `devbox update`, packages with the latest tag will be updated to the latest version available in our index.

## Manually Pinning a Nixpkg Commit for a Package

If you want to use a specific Nixpkg revision for a package, you can use a `github:nixos/nixpkgs/<commit_sha>#<pkg>` Flake reference. The example below shows how to install the `hello` package from a specific Nixpkg commit:

```json
{
  "packages" : [
    "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello"
  ]
}
```
Using multiple nixpkg commits may install duplicate packages and cause Nix Store bloat, so use this option sparingly.

## Pinning the Default Nixpkg commit in your Devbox.json

:::caution
Pinning the nixpkgs commit is considered deprecated starting with Devbox version 0.5.0, and will eventually be removed. We recommend using the `@` syntax to pin specific package versions instead.
:::

Devbox stores a default Nixpkg commit in your project's `devbox.json`, under the `nixpkgs.commit`. If you do not provide one yourself, Devbox will automatically add a default commit when you run a command like `devbox add`, `devbox shell`, or `devbox run`:

```json
"nixpkgs": {
    "commit": "89f196fe781c53cb50fef61d3063fa5e8d61b6e5"
}
```
This hash ensures that Devbox will install the same packages whenever you start a shell. By checking this into source control, you can ensure that other developers who run your project will get the same packages.
