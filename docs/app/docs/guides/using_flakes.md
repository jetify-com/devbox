---
title: Installing Packages from Nix Flakes
---

Devbox supports installing packages with [Nix Flakes](https://nixos.wiki/wiki/Flakes).

Devbox currently provides two ways to use Flakes to install packages in your project:

1. You can reference a Flake hosted in Github using the `github:` reference
2. You can reference a local Flake using the `path:` reference

## What are Flakes?

[Flakes](https://www.jetify.com/blog/powered-by-flakes/) are a new feature in the Nix language that lets you package software and create development shells in a declarative, fully reproducible way. You can use Nix Flakes to define packages, apps, templates, and dev environments.

Flakes are defined as a directory with a `flake.nix` and a `flake.lock` file. You import flakes to your project using a flake reference, which describes where to find the Flake, and what version or revision to use

## Using a Flake from Github

You can add a Flake hosted on Github using the following string in your packages list:

```json
"packages": [
    "github:<org>/<repo>/<ref>#<optional_flake_attr>"
]
```

The Ref and Flake Attribute is optional and will default to the main branch and `packages.default|defaultPackage` attribute, respectively.

For example, to install [Process Compose](https://github.com/F1bonacc1/process-compose) from its repository using Nix Flakes, you can use the following string in your packages list. This will install the latest version of Process Compose from the `main` branch.

```nix
github:F1bonacc1/process-compose
```

### Installing a Flake from a specific branch or tag

You can install a specific release or branch by adding it to your flake reference. The following example will install Process Compose version 0.40.2 from the `v0.40.2` tag.

```nix
github:F1bonacc1/process-compose/v0.40.2
```

### Installing a specific attribute or package from a Flake

You can also install a specific attribute or package from a Flake by adding a `#` and the attribute name to the end of the package string. If you don't specify an attribute, Devbox will use `default` or `defaultPackage`

For example, if you want to use [Fenix](https://github.com/nix-community/fenix) to install a specific version of Rust, you can use the following string in your packages list. This example will install the `stable.toolchain` packages from the `fenix` package.

```nix
github:nix-community/fenix#stable.toolchain
```

### Using Flakes with Nixpkgs

The Nixpkgs repo on Github also provides a Flake for installing packages. You can use the following flake reference to install packages from a specific Nixpkgs commit or reference:

```nix
github:NixOS/nixpkgs/<ref>#<package>
```

For example, if you want to install the `hello` package from the `nixos-20.09` branch, you can use the following string in your packages list:

```nix
github:NixOS/nixpkgs/nixos-20.09#hello
```

## Installing Additional Outputs from a Flake

Some packages provide additional outputs that are not installed by default. For example, the `libcap` package provides a `dev` output that contains development headers and libraries, or the `prometheus` package includes the `promtool` CLI in a `cli` output.

You can install these additional outputs by adding a `^` and a comma-separated list of outputs to the end of your flake reference. For example, the following command will install the default (`out`) and `dev` outputs of the `libcap` package:

```nix
github:nixos/nixpkgs#libcap^out,dev
```

## Using a Local Flake

You can also use a local Flake using the `path` attribute in your package list. Using a local flake can be helpful if you want to install your custom packages with Nix, or if you need to modify packages before using them in your Devbox project

Your flake reference should point to a directory that contains a `flake.nix` file.

```nix
path:<path_to_flake>#<optional_flake_attr>
```

For example, if you have a local Flake in the `./my-flake` directory, you can use the following string in your `packages` list. This example will install all the packages under the `my-package` attribute.

```nix
path:./my-flake#my-package
```

### Examples

For more examples of using Nix Flakes with Devbox, check out the examples in our Devbox Repo:

- [Using Nix Flakes from Github](https://github.com/jetify-com/devbox/tree/main/examples/flakes/remote)
- [Using a Local Flake](https://github.com/jetify-com/devbox/tree/main/examples/flakes/php)
- [Applying an Overlay with Nix Flakes](https://github.com/jetify-com/devbox/tree/main/examples/flakes/overlay)
