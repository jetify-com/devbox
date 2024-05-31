---
title: direnv
---


## direnv
___
[direnv](https://direnv.net) is an open source environment management tool that allows setting unique environment variables per directory in your file system. This guide covers how to configure direnv to seamlessly work with a devbox project.

:::note
Devbox 0.5.0 makes changes to how the environment is sourced in order to ensure better compatibility with the user's host shell. This may raise some errors if you generated your `.envrc` file with an older version of devbox.

If you see any errors when activating your `.envrc` file, you will need to run `devbox generate direnv --force`, and then re-run `devbox shell` to apply the latest changes. Be sure to back up your old `.envrc` file before running this command.

Direnv only supports modifying environment variables, so your `init_hook` functionality will be restricted. In particular, aliases, functions, and command completions will not work. If you use these, stick with the manual `devbox shell`.
:::

### Prerequisites
* Install direnv and hook it to your shell. Follow [this guide](https://direnv.net/#basic-installation) if you haven't done it.

### Setting up Devbox Shell and direnv

#### New Project

If you have direnv installed, Devbox will generate an .envrc file when you run `devbox generate direnv` and enables it by running `direnv allow` in the background:

```bash
➜  devbox generate direnv
Success: generated .envrc file
Success: ran `direnv allow`
direnv: loading ~/src/devbox/docs/.envrc
direnv: using devbox
```

This will generate a `.envrc` file in your project directory that contains `devbox.json`. Run `direnv allow` to activate your shell upon directory navigation. Run `direnv revoke` to stop. Changes to `devbox.json` automatically trigger direnv to reset the environment. The generated `.envrc` file doesn't need any further configuration. Just having the generated file along with installed direnv and Devbox, is enough to make direnv integrate with Devbox work.


#### Existing Project

For an existing project, you can add a `.envrc` file by running `devbox generate direnv`:

```bash
➜  devbox generate direnv
Success: generated .envrc file
Success: ran `direnv allow`
direnv: loading ~/src/devbox/docs/.envrc
direnv: using devbox
```

The generated `.envrc` file doesn't need any further configuration. Just having the generated file along with installed direnv and Devbox, is enough to make direnv integration with Devbox work.

#### Adding Custom Env Variables or Env Files to your Direnv Config

In some cases, you may want to override certain environment variables in your Devbox config when running it locally. You can add custom environment variables from the command line or from a file using the `--env` and `--env-file` flags.

If you would like to add custom environment variables to your direnv config, you can do so by passing the `--env` flag to `devbox generate direnv`. This flag takes a comma-separated list of key-value pairs, where the key is the name of the environment variable and the value is the value of the environment variable. For example, if you wanted to add a `MY_CUSTOM_ENV_VAR` environment variable with a value of `my-custom-value`, you would run the following command:

```bash
devbox generate direnv --env MY_CUSTOM_ENV_VAR=my-value
```

The resulting .envrc will have the following:

```bash
# Automatically sets up your devbox environment whenever you cd into this
# directory via our direnv integration:

eval "$(devbox generate direnv --print-envrc --env MY_CUSTOM_ENV_VAR=my-value)"

# check out https://www.jetify.com/devbox/docs/ide_configuration/direnv/
# for more details
```

You can also tell direnv to read environment variables from a custom `.env` file by passing the `--env-file` flag to `devbox generate direnv`. This flag takes a path to a file containing environment variables to set in the devbox environment. If the file does not exist, then this parameter is ignored. For example, if you wanted to add a `.env.devbox` file located in your project root, you would run the following command:

```bash
devbox generate direnv --env-file .env.devbox
```

The resulting .envrc will have the following:

```bash
# Automatically sets up your devbox environment whenever you cd into this
# directory via our direnv integration:

eval "$(devbox generate direnv --print-envrc --env-file .env.devbox)"

# check out https://www.jetify.com/devbox/docs/ide_configuration/direnv/
# for more details
```

Note that if Devbox cannot find the env file provided to the flag, it will ignore the flag and load your Devbox shell environment as normal

### Global settings for direnv

Note that every time changes are made to `devbox.json` via `devbox add ...`, `devbox rm ...` or directly editing the file, requires `direnv allow` to run so that `direnv` can setup the new changes.

Alternatively, a project directory can be whitelisted so that changes will be automatically picked up by `direnv`. This is done by adding following snippet to direnv config file typically at `~/.config/direnv/direnv.toml`. You can create the file and directory if it doesn't exist.

```toml
[whitelist]
prefix = [ "/absolute/path/to/project" ]

```

### Direnv Limitations

Direnv works by creating a sub-shell using your `.envrc` file, your `devbox.json`, and other direnv related files, and then exporting the diff in environment variables into your current shell. This imposes some limitations on what it can load into your shell: 

1. Direnv cannot load shell aliases or shell functions that are sourced in your project's `init_hook`. If you want to use direnv and also configure custom aliases, we recommend using [Devbox Scripts](../guides/scripts.md). 
2. Direnv does not allow modifications to the $PS1 environment variable. This means `init_hooks` that modify your prompt will not work as expected. For more information, see the [direnv wiki](https://github.com/direnv/direnv/wiki/PS1)

Note that sourcing aliases, functions, and `$PS1` should work as expected when using `devbox shell`, `devbox run`, and `devbox services`

### VSCode setup with direnv

To seamlessly integrate VSCode with a direnv environment, follow these steps:

1. Open a terminal window and activate direnv with `direnv allow`.
2. Launch VSCode from the same terminal window using the command `code .` This ensures that VSCode inherits the direnv environment.

Alternatively, you can use the [direnv VSCode extension](https://marketplace.visualstudio.com/items?itemName=mkhl.direnv) if your VSCode workspace has a .envrc file.

If this guide is missing something, feel free to contribute by opening a [pull request](https://github.com/jetify-com/devbox/pulls) in Github.
