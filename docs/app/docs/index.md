---
title: Introduction
sidebar_position: 1
hide_title: true
---

## What is Devbox?
Devbox is a command-line tool that lets you easily create isolated shells and containers. You start by defining the list of packages required by your development environment, and devbox uses that definition to create an isolated environment just for your application.

In practice, Devbox works similar to a package manager like yarn – except the packages it manages are at the operating-system level (the sort of thing you would normally install with brew or apt-get).

<figure>

![screen cast](https://user-images.githubusercontent.com/279789/186491771-6b910175-18ec-4c65-92b0-ed1a91bb15ed.svg)

<figcaption>Create isolated dev environments on the fly with Devbox</figcaption>
</figure>

## Why Use Devbox?

Devbox provides a lot of benefits over pure Docker containers, Nix Shells, or managing your own environment directly: 

### A consistent shell for everyone on the team
Declare the list of tools needed by your project via a devbox.json file and run devbox shell. Everyone working on the project gets a shell environment with the exact same version of those tools.

### Try new tools without polluting your laptop
Development environments created by Devbox are isolated from everything else in your laptop. Is there a tool you want to try without making a mess? Add it to a Devbox shell, and remove it when you don't want it anymore – all while keeping your laptop pristine. Removing or changing a package in your dev environment is as easy as editing your `devbox.json`.

### Don't sacrifice speed
Devbox can create isolated environments right on your laptop, without an extra-layer of virtualization slowing your file system or every command. When you're ready to ship, it'll turn it into an equivalent container – but not before.

### Good-bye conflicting versions
Are you working on multiple projects, all of which need different versions of the same binary? Instead of attempting to install conflicting versions of the same binary on your laptop, create an isolated environment for each project, and use whatever version you want for each.

### Instantly turn your application into a container
Devbox analyzes your source code and instantly turns it into an OCI-compliant image that can be deployed to any cloud. The image is optimized for speed, size, security and caching ... and without needing to write a Dockerfile. And unlike buildpacks, it does it quickly.

### Stop declaring dependencies twice
Your application often needs the same set of dependencies when you are developing on your laptop, and when you're packaging it as a container ready to deploy to the cloud. Devbox's dev environments are isomorphic: meaning that we can turn them into both a local shell environment or a cloud-ready container, all without having to repeat yourself twice.
