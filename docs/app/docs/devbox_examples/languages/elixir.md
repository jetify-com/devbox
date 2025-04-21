---
title: Elixir
---

Elixir can be installed by simply running `devbox add elixir`. This will automatically include the Elixir Plugin to isolate Mix/Hex artifacts and enable shell history in `iex`.

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/elixir/elixir_hello)


## Adding Elixir to your project

`devbox add elixir`, or add the following to your `devbox.json`

```json
    "packages": [
        "elixir@latest"
    ],
```

This will install the latest version of Elixir. You can find other installable versions of Elixir by running `devbox search elixir`. You can also search for Elixir on [Nixhub](https://www.nixhub.io/packages/elixir)

## Elixir Plugin Support

Devbox will automatically use the following configuration when you install Elixir with `devbox add`.

### Environment Variables

`$MIX_HOME` and `$HEX_HOME` configure Mix/Hex to install artifacts locally, while `$ERL_AFLAGS` enables shell history in `iex`:

```bash
MIX_HOME={PROJECT_DIR}/.devbox/virtenv/elixir/mix
HEX_HOME={PROJECT_DIR}/.devbox/virtenv/elixir/hex
ERL_AFLAGS="-kernel shell_history enabled"
```

### Disabling the Elixir Plugin

You can disable the Elixir plugin by running `devbox add elixir --disable-plugin`, or by setting the `disable_plugin` field in your `devbox.json`:

```json
{
    "packages": {
        "elixir": {
            "version": "latest",
            "disable_plugin": true
        }
    },
}
```

Note that disabling the plugin will cause Mix and Hex to cache artifacts globally in the user's home directory (at `~/.mix/` and `~/.hex/`). This might actually be preferable if you're developing several Elixir projects and want to benefit from caching, but does defeat the isolation guarantees of Devbox.

If the plugin is disabled, it's recommended to manually set `$ERL_AFLAGS` to preserve `iex` shell history:

```json
    "env": {
      "ERL_AFLAGS": "-kernel shell_history enabled"
    }
```