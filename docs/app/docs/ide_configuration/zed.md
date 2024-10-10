---
title: Zed Editor
---

[Zed](https://zed.dev/) is a fast, open source code editor designed for collaboration and AI support, that is available for macOS and Linux. Zed has support for loading environments directly from Direnv's `.envrc` files, meaning you can easily use Zed w/ Devbox via our [direnv integration](/devbox/ide_configuration/direnv).

## Setting up your Project for Zed

1. Make sure that you have direnv installed on your host. To use direnv across all your projects, we recommend installing it with [devbox global](/devbox/devbox_global) using `devbox global add direnv`. You can also follow [this guide](https://direnv.net/#basic-installation) to configure direnv for your system

2. Generate a `.envrc` file for your project by running `devbox generate direnv` in your project's root directory (the same directory with your `devbox.json` file.

3. You can now open your project in Zed and it will automatically load your Devbox shell environment variables from the `.envrc` file.

## Troubleshooting your Zed Setup

If you are having trouble getting Zed's LSP to detect your Devbox environment, try the following steps:

1. Make sure you are up to date with the latest version of Zed. You can check for updates by going to `Zed > Check for Updates` in the Zed menu.

2. You may need to explicitly tell your LSP to use the binaries in your $PATH variable. To do this, add the following to the `~/.config/zed/config.json` file:

```json
{
  ...
  "lsp": {
    "<lsp-name>": {
      "binary": {"path_lookup": true}
    }
  },
  ...
}
```
