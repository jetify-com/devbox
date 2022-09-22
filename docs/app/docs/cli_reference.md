---
title: CLI Reference
sidebar_position: 4
---

## add

Add Nix packages to your devbox project. To see a list of available packages, you can use [Nix Package Search](https://search.nixos.org/packages). Added packages are stored in `devbox.json`

For example the following will add Python 3.10 to your `devbox.json`

```nix
devbox add python310
```

You can add multiple packages in a single command:

```nix
devbox add python310 curl 
```

## build

Builds your current source directory and devbox configuration as a Docker container. Devbox will create a plan for your container based on your source code, and then apply the packages and stage overrides in your `devbox.json`.

## init

Initializes a devbox project by creating a blank `devbox.json` file in your directory. 

## plan

Shows the current plan that devbox will use to create your shell and build your project. 

This will include any install/build/run commands that devbox has automatically detected, or that you have included in your devbox.json (see [Configuration Guide](../configuration) for more details)

## rm

Removes packages from your devbox.json.

## shell

Starts an isolated nix shell with the packages and configuration in your local `devbox.json`. 

Any packages in your devbox.json that have not been previously installed will be downloaded and installed in your machine, and made available while your shell is running. 
