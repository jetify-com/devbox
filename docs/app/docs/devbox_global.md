---
title: Use Devbox as your Primary Package Manager
description: Install packages and tools system wide with Devbox Global
---

In addition to managing isolated development environments, you can use Devbox as a general package manager. Devbox Global allows you to add packages to a global `devbox.json.` This is useful for installing a standard set of tools you want to use across multiple Devbox Projects.

For example — if you use ripgrep as your preferred search tool, you can add it to your global Devbox profile with `devbox global add ripgrep`. Now whenever you start a Devbox shell, you will have ripgrep available, even if it's not in the project's devbox.json.

<figure>

![Installing ripgrep using `devbox global add ripgrep](../static/img/devbox_global.svg)

<figcaption>Installing Packages with Devbox Global</figcaption>
</figure>

You can also use `devbox global` to replace package managers like `brew` and `apt` by adding the global profile to your path. Because Devbox uses Nix to install packages, you can sync your global config to install the same packages on any machine.

Devbox saves your global config in a `devbox.json` file in your home directory. This file can be shared with other users or checked into source control to synchronize it across machines.


## Adding and Managing Global Packages

You can install a package using `devbox global add [<package>]`, where the package names should be a list of [Nix Packages](https://search.nixos.org/packages) you want to install.

For example, if we wanted to install ripgrep, vim, and git to our global profile, we could run:

```bash
devbox global add ripgrep vim git

# Output:
ripgrep is now installed
vim is now installed
git is now installed
```

Once installed, the packages will be available whenever you start a Devbox Shell, even if it's not included in the project's `devbox.json`.

To view a full list of global packages, you can run `devbox global list`:

```bash
devbox global list

# Output:
* ripgrep
* vim
* git
```

To remove a global package, use:

```bash
devbox global rm ripgrep

# Output:
removing 'github:NixOS/nixpkgs/ripgrep'
```

## Using Fleek with Devbox Global

[Fleek](https://getfleek.dev/) provides a nicely tuned set of packages and configuration for common tools that is compatible with Devbox Global. Configurations are provided at different [levels of bling](https://getfleek.dev/docs/bling), with higher levels adding more packages and opinionated configuration.

To install a Fleek profile, you can use `devbox global pull <fleek-url>`, where the Fleek URL indicates the profile you want to install. For example, to install the `high` bling profile, you can run:

```bash
devbox global pull https://devbox.getfleek.dev/high
```

Fleek profiles also provide a few convenience scripts to automate setting up your profile. You can view the full list of scripts using `devbox global run` with no arguments

For more information, see the [Fleek for Devbox Docs](https://getfleek.dev/docs/devbox)

## Using Global Packages in your Host Shell

If you want to make your global packages available in your host shell, you can add them to your shell PATH. Running `devbox global shellenv` will print the command necessary to source the packages.

### Add Global Packages to your Current Host Shell
To temporarily add the global packages to your current shell, run:

```bash
. <(devbox global shellenv --init-hook)
```

You can also add a hook to your shell's config to make them available whenever you launch your shell:

### Bash

Add the following command to your `~/.bashrc` file:

```bash
eval "$(devbox global shellenv --init-hook)"
```

Make sure to add this hook before any other hooks that use your global packages.

### Zsh
Add the following command to your `~/.zshrc` file:

```bash
eval "$(devbox global shellenv --init-hook)"
```

### Fish

Add the following command to your `~/.config/fish/config.fish` file:

```bash
devbox global shellenv --init-hook | source
```

## Sharing Your Global Config with Git

You can use Git to synchronize your `devbox global` config across multiple machines using `devbox global push <remote>` and `devbox global pull <remote>`.

Your global `devbox.json` and any other files in the Git remote will be stored in `$XDG_DATA_HOME/devbox/global/default`. If `$XDG_DATA_HOME` is not set, it will default to `~/.local/share/devbox/global/default`. You can view the current global directory by running `devbox global path`.

## Next Steps

### Learn more about Devbox

* **[Getting Started](quickstart.mdx):** Learn how to install Devbox and create your first Devbox Shell.
* **[Devbox Scripts](guides/scripts.md):** Automate setup steps and configuration for your shell using Devbox Scripts.
* **[Configuration Guide](configuration.md):** Learn how to configure your shell and dev environment with `devbox.json`.
* **[Browse Examples](https://github.com/jetify-com/devbox-examples):** You can see how to create a development environment for your favorite tools or languages by browsing the Devbox Examples repo.
* **[Using Flakes with Devbox](guides/using_flakes.md):** Learn how to install packages from Nix Flakes.

### Use Devbox with your IDE

* **[Direnv Integration](ide_configuration/direnv.md):** Devbox can integrate with [direnv](https://direnv.net/) to automatically activate your shell and packages when you navigate to your project.
* **[Devbox for Visual Studio Code](https://marketplace.visualstudio.com/items?itemName=jetify-com.devbox):** Install our VS Code extension to speed up common Devbox workflows or to use Devbox in a devcontainer.

### Get Involved

* **[Join our Discord Community](https://discord.gg/jetify):** Chat with the development team and our growing community of Devbox users.
* **[Visit us on Github](https://github.com/jetify-com/devbox):** File issues and provide feedback, or even open a PR to contribute to Devbox or our Docs.
