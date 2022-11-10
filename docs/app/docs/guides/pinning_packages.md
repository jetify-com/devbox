---
title: Pinning Packages with Nixpkg
---

This doc will explain how to select and pin specific package versions in Devbox by setting a Nixpkg commit in your devbox.json

## Background

The Nix Package Manager, which Devbox uses to install your shell packages, stores it's package definitions in a Github Repository at [NixOS/nixpkgs](https://github.com/NixOS/nixpkgs). This repository contains the Nix build definitions for over 80,000 different packages. New packages or changes to build definitions are added (and deprecated packages are removed) by committing to the repo. 

Because Nix uses Git to store it's package definitions, we can install specific packages from older versions of the Nix Store by specifying the commit that we want to use. You can also use this commit to pin your project to a specific version of Nixpkgs, so that any developer using your project will get the exact same packages. 

## Pinning the Nixpkg commit in your Devbox.json

Devbox stores the Nixpkg commit in your project's `devbox.json`, under the `nixpkgs.commit`. If you do not specify one in your config, Devbox will automatically add a default commit hash when you run a command like `devbox add`, `devbox shell`, or `devbox run`:

```json
"nixpkgs": {
    "commit": "89f196fe781c53cb50fef61d3063fa5e8d61b6e5"
}
```
This hash ensures that Devbox will install the same packages whenever you start a shell. By checking this into source control, you can also ensure that any other developers who run your project will get the same packages.

## Using the latest version of Nixpkgs

To use the latest available packages in Nix, you can replace the commit in `devbox.json` with the latest `nixpkgs-unstable` hash from https://status.nixos.org. 

## Look up a commit hash for a specific package

In most cases, the packages available in Devbox's default commit should suffice for your use cases. However, if you need a specific minor version, or an older version of a package that is no longer included in Nixpkgs, you may need update the commit SHA. Unfortunately, Nix does not have an official way to find the Nixpkg commit SHA for a specific version of a package. 

However, there is an unofficial search tool at https://lazamar.co.uk/nix-versions/ that can be used to list the Nixpkg commits for different versions of a specific package. To find the correct Nixpkg commit hash: 
1. Select `nixpkgs-unstable` in the dropdown
2. Enter the name of the package you want to search, and hit Search
3. In the search results, find the version you want in the Version Column
4. Copy the Commit hash in the Revision column
5. Add the commit hash to your `devbox.json`
