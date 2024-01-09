---
title: Frequently Asked Questions
description: Frequently Asked Questions about Devbox
---

## How does Devbox work?

Devbox generates isolated, reproducible development environments using the [Nix package manager](https://nixos.org/). Devbox uses Nix to install packages, and then creates an isolated shell environment for your project by symlinking the packages you need into your project directory.

## Where does Devbox install my packages?

Devbox and Nix install your packages in the read-only Nix store, usually located at `/nix/store`. Devbox then creates your environment by symlinking the packages you need into the `.devbox` directory in your project.

## How can I delete packages from the Nix Store that I no longer need?

You can use `nix store gc` to automatically clean up packages that are no longer needed for your projects.

## Does Devbox require Docker or Containers to work?

No. Since Devbox uses Nix to install packages and create isolated environments, Docker is not required. If you want to run your Devbox project inside a container, you can generate a Dockerfile or devcontainer.json using the `devbox generate` command.

## What versions of Nix are supported by Devbox?

Devbox requires Nix >= 2.12. If Nix is not present on your machine when you first run Devbox, it will automatically try to install the latest supported version for you.

## Can I use Devbox alongside NixOS?

Yes! Devbox can be installed on any Linux distribution, including NixOS. You can even install Devbox via Nixpkgs. See the [installation guide](./installing_devbox.mdx) for more details.

## I installed a package that I need for development, but it’s missing header files/libraries/other components that I was expecting, where can I find these?

In order to save space, Devbox and Nix only install the required components of packages by default. Development header files and libraries are often installed in a separate output of the package (usually `dev`), which can be installed using [Flake References](./guides/using_flakes.md).

You can learn more about non-default outputs [here](./guides/using_flakes.md#installing-additional-outputs-from-a-flake)

## I have some custom packages or nix expressions that I’ve written. How can I integrate those into Devbox?

You can add customized packages to your Devbox environment using our [Flake support](./guides/using_flakes.md). You can use these flakes to modify or override packages from nixpkgs, or to create your own custom packages.

## Can I use Devbox if I use the [Fish](https://fishshell.com/) shell?

Yes. In addition to supporting POSIX compliant shells like Zsh and Bash, Devbox also works with Fish.

## How can I uninstall Devbox?

To uninstall Devbox:

1. Remove the Devbox launcher using `rm /usr/local/bin/devbox`
2. Remove the Devbox binaries using `rm -rf ~/.cache/devbox`
3. Remove your Devbox global config using `rm -rf .local/share/devbox`

If you want to uninstall Nix, you will need to follow the instructions in the Nix Documentation: https://nixos.org/manual/nix/stable/installation/uninstall.