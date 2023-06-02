---
title: direnv 
---


## direnv
___
[direnv](https://direnv.net) is an open source environment management tool that allows setting unique environment variables per directory in your file system. This guide covers how to configure direnv to seamlessly work with a devbox project.

:::note 
Devbox 0.5.0 makes changes to how the environment is sourced in order to ensure better compatibility with the user's host shell. This may raise some errors if you generated your `.envrc` file with an older version of devbox.
    
If you see any errors when activating your `.envrc` file, you will need to run `devbox generate direnv --force`, and then re-run `devbox shell` to apply the latest changes. Be sure to back up your old `.envrc` file before running this command.
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

This will generate a `.envrc` file in your project directory that contains `devbox.json`. Run `direnv allow` to activate your shell upon directory navigation. Run `direnv revoke` to stop. Changes to `devbox.json` automatically trigger direnv to reset the environment.


#### Existing Project

For an existing project, you can add a `.envrc` file by running `devbox generate direnv`:

```bash
➜  devbox generate direnv
Success: generated .envrc file
Success: ran `direnv allow`
direnv: loading ~/src/devbox/docs/.envrc
direnv: using devbox
```

### Global settings for direnv

Note that every time changes are made to `devbox.json` via `devbox add ...`, `devbox rm ...` or directly editing the file, requires `direnv allow` to run so that `direnv` can setup the new changes.

Alternatively, a project directory can be whitelisted so that changes will be automatically picked up by `direnv`. This is done by adding following snippet to direnv config file typically at `~/.config/direnv/direnv.toml`. You can create the file and directory if it doesn't exist.

```toml
[whitelist]
prefix = [ "/absolute/path/to/project" ]

```

### VSCode setup with direnv

To seamlessly integrate VSCode with a direnv environment, follow these steps:

1. Open a terminal window and activate direnv with `direnv allow`.
2. Launch VSCode from the same terminal window using the command `code .` This ensures that VSCode inherits the direnv environment.

Alternatively, you can use the [direnv VSCode extension](https://marketplace.visualstudio.com/items?itemName=mkhl.direnv) if your VSCode workspace has a .envrc file.

If this guide is missing something, feel free to contribute by opening a [pull request](https://github.com/jetpack-io/devbox/pulls) in Github.
