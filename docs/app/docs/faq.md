---
title: Frequently Asked Questions
description: Frequently Asked Questions about Devbox
---

This doc contains answers to frequently asked questions about Devbox that are not covered elsewhere in our documentation. If you have a question that isn't covered here, feel free to ask us on our [Discord](https://discord.gg/jetpack-io), or [open an issue](https://github.com/jetpack-io/devbox/issues) on our GitHub repository.

## How does Devbox work?

Devbox generates isolated, reproducible development environments using the [Nix package manager](https://nixos.org/). Devbox uses Nix to install packages, and then creates an isolated shell environment for your project by symlinking the packages you need into your project directory.

## Where does Devbox install my packages?

Devbox and Nix install your packages in the read-only Nix store, usually located at `/nix/store`. Devbox then creates your environment by symlinking the packages you need into the `.devbox` directory in your project.

## How do I clean up unused packages from the Nix Store?

You can use `devbox run -- nix store gc` to automatically clean up packages that are no longer needed for your projects.

## Does Devbox require Docker or Containers to work?

No. Since Devbox uses Nix to install packages and create isolated environments, Docker is not required. If you want to run your Devbox project inside a container, you can generate a Dockerfile or devcontainer.json using the `devbox generate` command.

## What versions of Nix are supported by Devbox?

Devbox requires Nix >= 2.12. If Nix is not present on your machine when you first run Devbox, it will automatically try to install the latest supported version for you.

## Can I use Devbox with NixOS?

Yes! Devbox can be installed on any Linux distribution, including NixOS. You can even install Devbox via Nixpkgs. See the [installation guide](./installing_devbox.mdx) for more details.

## A package I installed is missing header files or libraries I need for development. Where do I find them?

In order to save space, Devbox and Nix only install the required components of packages by default. Development header files and libraries are often installed in a separate output of the package (usually `dev`), which can be installed using [Flake References](./guides/using_flakes.md).

You can learn more about non-default outputs [here](./guides/using_flakes.md#installing-additional-outputs-from-a-flake).

## I'm trying to build a project, but it says that I'm missing `libstdc++`. How do I install this library in my project?

This message means that your project requires an implementation of the C++ Standard Library installed and linked within your shell. You can add the libstdc++ libraries and object files using `devbox add stdenv.cc.cc.lib`. 

## How can I use custom Nix packages or overrides with Devbox?

You can add customized packages to your Devbox environment using our [Flake support](./guides/using_flakes.md). You can use these flakes to modify or override packages from nixpkgs, or to create your own custom packages.

## Can I use Devbox if I use [Fish](https://fishshell.com/)?

Yes. In addition to supporting POSIX compliant shells like Zsh and Bash, Devbox also works with Fish. 

Note that `init_hooks` in Devbox will be run directly in your host shell, so you may have encounter some compatibility issues if you try to start a shell that uses a POSIX-compatible script in the init_hook.  

## How can I rollback to a previous version of Devbox?

You can use any previous version of Devbox by setting the `DEVBOX_USE_VERSION` environment variable. For example, to use version 0.8.0, you can run the following or add it to your shell's rcfile: 

```bash
export DEVBOX_USE_VERSION=0.8.0
```

You can upgrade to the latest version of Devbox by unsetting the variable, and running `devbox version update`

## How can I prevent Devbox from modifying my prompt while inside a shell?

By default, Devbox will prefix your prompt with `(devbox)` when inside a `devbox shell`. You can disable this behavior by setting this environment variable in your shell's rcfile:

```bash
DEVBOX_NO_PROMPT=true
```

You can now detect being inside a `devbox shell` and change your prompt using the method of your choosing.

## How can I uninstall Devbox?

To uninstall Devbox:

1. Remove the Devbox launcher using `rm /usr/local/bin/devbox`
2. Remove the Devbox binaries using `rm -rf ~/.cache/devbox`
3. Remove your Devbox global config using `rm -rf .local/share/devbox`

If you want to uninstall Nix, you will need to follow the instructions in the Nix Documentation: https://nixos.org/manual/nix/stable/installation/uninstall.
