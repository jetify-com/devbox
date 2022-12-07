---
title: Elixir
---

Elixir can be configured to install Hex and Rebar dependencies in a local directory. This will keep Elixir from trying to install in your immutable Nix Store: 

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/development/elixir/elixir_hello)

## Adding Elixir to your project

`devbox add elixir bash`, or add the following to your `devbox.json`

```json
    "packages": [
        "elixir",
        "bash"
    ],
```

This will install Elixir 1.13. 
Other versions available include: 

* elixir_1_10 (version 1.10)
* elixir_1_11 (version 1.11)
* elixir_1_12 (version 1.12)
* elixir_1_14 (version 1.14)

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