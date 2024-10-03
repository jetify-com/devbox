---
title: Elixir
---

Elixir can be configured to install Hex and Rebar dependencies in a local directory. This will keep Elixir from trying to install in your immutable Nix Store:

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/elixir/elixir_hello)

[![Open In Devspace](../../../static/img/open-in-devspace.svg)](https://www.jetify.com/devbox/templates/elixir/)

## Adding Elixir to your project

`devbox add elixir bash`, or add the following to your `devbox.json`

```json
    "packages": [
        "elixir@latest",
        "bash@latest"
    ],
```

This will install the latest version of Elixir available. You can find other installable versions of Elixir by running `devbox search elixir`. You can also search for Elixir on [Nixhub](https://www.nixhub.io/packages/elixir)

## Installing Hex and Rebar locally

Since you are unable to install Elixir Deps directly into the Nix store, you will need to configure mix to install your dependencies globally. You can do this by adding the following lines to your `devbox.json` init_hook:

```json
    "shell": {
        "init_hook": [
            "mkdir -p .nix-mix",
            "mkdir -p .nix-hex",
            "export MIX_HOME=$PWD/.nix-mix",
            "export HEX_HOME=$PWD/.nix-hex",
            "export ERL_AFLAGS='-kernel shell_history enabled'",
            "mix local.hex --force",
            "mix local.rebar --force"
        ]
    }
```

This will create local folders and force mix to install your Hex and Rebar packages to those folders. Now when you are in `devbox shell`, you can install using `mix deps`.
