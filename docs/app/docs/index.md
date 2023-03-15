---
title: Introduction
sidebar_position: 1
hide_title: true
---

## What is Devbox?
Devbox is a command-line tool that lets you easily create isolated shells for development. You start by defining the list of packages required for your project, and Devbox creates an isolated, reproducible environment with those packages installed.

In practice, Devbox works similar to a package manager like yarn – except the packages it manages are at the operating-system level (the sort of thing you would normally install with brew or apt-get).

<figure>

![screen cast](https://user-images.githubusercontent.com/279789/186491771-6b910175-18ec-4c65-92b0-ed1a91bb15ed.svg)

<figcaption>Create isolated dev environments on the fly with Devbox</figcaption>
</figure>

## Why Use Devbox?

Devbox provides a lot of benefits over pure Docker containers, Nix Shells, or managing your own environment directly: 

### A consistent shell for everyone on the team
Declare the list of tools needed by your project via a `devbox.json` file and run devbox shell. Everyone working on the project gets a shell environment with the exact same version of those tools.

### Try new tools without polluting your laptop
Development environments created by Devbox are isolated from everything else in your laptop. Is there a tool you want to try without making a mess? Add it to a Devbox shell, and remove it when you don't want it anymore – all while keeping your laptop pristine. Removing or changing a package in your dev environment is as easy as editing your `devbox.json`.

### Don't sacrifice speed
Devbox can create isolated environments right on your laptop, without an extra-layer of virtualization slowing your file system or every command. When you're ready to ship, it'll turn it into an equivalent container – but not before.

### Good-bye conflicting versions
Are you working on multiple projects, all of which need different versions of the same binary? Instead of attempting to install conflicting versions of the same binary on your laptop, create an isolated environment for each project, and use whatever version you want for each.

### Take your environment with you
Devbox's dev environments are _portable_. We make it possible to declare your environment exactly once, and use that single definition in several different ways, including:
+ A local shell created through `devbox shell`
+ A devcontainer you can use with VSCode
+ A Dockerfile so you can build a production image with the exact same tools you
  used for development.
+ A remote development environment in the cloud that mirrors your local environment.
