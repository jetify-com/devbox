---
title: "Starting a Dev Environment with Devbox"
sidebar_position: 3
---
## Background

Devbox is a command-line tool that lets you easily create reproducible, reliable dev environments. You start by defining the list of packages required by your development environment, and devbox uses that definition to create an isolated environment just for your application. Developers can start a dev environment for their project by running `devbox shell`.

To learn more about how Devbox works, you can read our [introduction](index.md)

This quickstart shows you how to install Devbox, and use it to start a development environment for a project that is configured to use Devbox via `devbox.json`


## Install Devbox

Use the following install script to get the latest version of Devbox:

```bash
curl -fsSL https://get.jetify.com/devbox | bash
```

Devbox requires the [Nix Package Manager](https://nixos.org/download.html). If Nix is not detected on your machine when running a command, Devbox will automatically install it for you with the default settings for your OS. Don't worry: You can use Devbox without needing to learn the Nix Language.

## Start your development shell

1. Open a terminal in the project. The project should contain a `devbox.json` that specifies how to create your development environment

1. Start a devbox shell for your project:

    To get started, all we have to do is run:
    ```bash
    devbox shell
    ```

    **Output:**
    ```bash
    Installing nix packages. This may take a while... done.
    Starting a devbox shell...
    (devbox) $
    ```

    :::info
    The first time you run `devbox shell` may take a while to complete due to Devbox downloading prerequisites and package catalogs required by Nix. This delay is a one-time cost, and future invocations and package additions should resolve much faster.
    :::

1. Use the packages provided in your development environment

    The packages listed in your project's `devbox.json` should now be available for you to use. For example, if the project's `devbox.json` contains `python@3.10`, you should now have `python` in your path:

    ```bash
    $ python --version
    Python 3.10.9
    ```

1. Your host environment's packages and tools are also available, including environment variables and config settings.

    ```bash
    git config --get user.name
    ```

1. You can search for additional packages using `devbox search <pkg>`. You can then add them to your Devbox shell by running `devbox add [pkgs]`

1. To exit the Devbox shell and return to your regular shell:

    ```bash
    exit
    ```

## Next Steps

### Learn more about Devbox
* **[Devbox Global](devbox_global.md):** Learn how to use the devbox as a global package manager
* **[Devbox Scripts](guides/scripts.md):** Automate setup steps and configuration for your shell using Devbox Scripts.
* **[Configuration Guide](configuration.md):** Learn how to configure your shell and dev environment with `devbox.json`.
* **[Browse Examples](https://github.com/jetify-com/devbox-examples):** You can see how to create a development environment for your favorite tools or languages by browsing the Devbox Examples repo.

### Use Devbox with your IDE
* **[Direnv Integration](ide_configuration/direnv.md):** Devbox can integrate with [direnv](https://direnv.net/) to automatically activate your shell and packages when you navigate to your project.
* **[Devbox for Visual Studio Code](https://marketplace.visualstudio.com/items?itemName=jetpack-io.devbox):** Install our VS Code extension to speed up common Devbox workflows or to use Devbox in a devcontainer.

### Get Involved
* **[Join our Discord Community](https://discord.gg/jetify):** Chat with the development team and our growing community of Devbox users.
* **[Visit us on Github](https://github.com/jetify-com/devbox):** File issues and provide feedback, or even open a PR to contribute to Devbox or our Docs.
