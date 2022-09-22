---
title: Installing Devbox
sidebar_position: 2
---

## Prerequisities

In addition to installing Devbox itself, you will need to install nix and docker since Devbox depends on them:

1. Install [Nix Package Manager](https://nixos.org/download.html). (Don't worry, you don't need to learn Nix.)

2. Install [Docker Engine](https://docs.docker.com/engine/install/) or [Docker Desktop](https://www.docker.com/get-started/). Note that docker is only needed if you want to create containers – the shell functionality works without it.

## Install Devbox

Use the following install script to get the latest version of Devbox:

```bash
curl -fsSL https://get.jetpack.io/devbox | bash
```
