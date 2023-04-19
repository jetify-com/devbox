---
title: Pinning Packages with Nixpkg
---

This doc will explain how to select and pin specific package versions in Devbox by setting a Nixpkg commit in your devbox.json

## Background

The Nix Package Manager, which Devbox uses to install your shell packages, stores its package definitions in a Github Repository at [NixOS/nixpkgs](https://github.com/NixOS/nixpkgs). This repository contains instructions for building over 80,000 different packages. Maintainers add new packages or remove deprecated packages by committing to the repo. 

Because Nix uses Git to store its package definitions, we can install specific packages from older versions of the Nix Store by specifying the default commit we want to use. You can also use this commit to pin your project to a specific version of Nixpkgs, so any developer using your project will get the same packages. 


## Pinning the Default Nixpkg commit in your Devbox.json

Devbox stores the Nixpkg commit in your project's `devbox.json`, under the `nixpkgs.commit`. If you do not specify one in your config, Devbox will automatically add a default commit hash when you run a command like `devbox add`, `devbox shell`, or `devbox run`:

```json
"nixpkgs": {
    "commit": "89f196fe781c53cb50fef61d3063fa5e8d61b6e5"
}
```
This hash ensures that Devbox will install the same packages whenever you start a shell. By checking this into source control, you can also ensure that any other developers who run your project will get the same packages.

## Using the latest version of Nixpkgs

To use the latest available packages in Nix, you can replace the commit in `devbox.json` with the latest `nixpkgs-unstable` hash from [https://status.nixos.org](https://status.nixos.org). 

## Pinning a Nixpkg Commit for a Single Package

If you want to use a different commit for a single package, you can use a Flake reference to use an older revision of Nixpkg for just that package. The example below shows how to install the `hello` package from a specific Nixpkg commit:

```json
}
	"packages" : [
"github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello"
	]
}
```
Note that using a different nixpkg commit may install some duplicate packages and cause Nix Store bloat, so use this option sparingly. 

## How to Find the Nixpkg Commit for a Package

In most cases, the packages available in Devbox's default commit should suffice for your use cases. However, if you want to install an older package no longer available in Nix, you must use an older commit reference in either your Flake reference or default nixpkg commit.

Unfortunately, Nix does not have an official way to find the Nixpkg commit SHA for a specific package. However, an unofficial search tool at [https://lazamar.co.uk/nix-versions/](https://lazamar.co.uk/nix-versions/) can be used to list the Nixpkg commits for different versions of a specific package. To find the correct Nixpkg commit hash: 
1. Select `nixpkgs-unstable` in the dropdown
2. Enter the name of the package you want to search, and hit Search
3. In the search results, find the version you want in the Version Column
4. Copy the commit hash in the Revision column
5. Add the commit hash to your `devbox.json`
