---
title: Configuration Guide
sidebar_position: 5
---

Your devbox configuration is stored in a `devbox.json` file, located in your project's root directory. This file can be edited directly, or using the [devbox CLI](cli_reference/devbox.md).

```json
{
    "packages": [],
    "shell": {
        "init_hook": "..."
    },
    "install_stage": {
        "command": "..."
    },
    "build_stage": {
        "command": "..."
    },
    "start_stage": {
        "command": "..."
    }
}
```

### Packages

This is a list of Nix packages that should be installed in your Devbox shell and containers. These packages will only be installed and available within your shell, and will have precedence over any packages installed in your local machine. You can search for Nix packages using [Nix Package Search](https://search.nixos.org/packages).

You can add packages to your devbox.json using `devbox add <package_name>`, and remove them using `devbox rm <package_name>`

### Shell

You can configure `devbox shell` to run a custom commands at startup by setting an `init_hook`. This hook runs after any other `~/.*rc` scripts, allowing you to override environment variables or further customize the shell.

This is an example `devbox.json` that customizes the prompt and prints a welcome message:

```json
{
    "shell": {
        "init_hook": "export PS1='📦 devbox> '\necho 'Welcome! See CONTRIBUTING.md for tips on contributing to devbox.'"
    }
}
```

When run, you'll see:

```text
> devbox shell
Installing nix packages. This may take a while...
Starting a devbox shell...
Welcome! See CONTRIBUTING.md for tips on contributing to devbox.
📦 devbox>
```

### Stages

Stages are used to configure and run commands at different points of container creation. For languages that support autodetction, Devbox will automatically detect and configure the correct stage commands for your project based on your source code. You can override any of these stages by configuring them in your devbox.json

-   The **install stage** will run after your base container has been initialized and your Nix packages are installed. This stage should be used to download and build your application's dependencies
-   The **build stage** runs after the install stage, and should be used to build or bundle your application.
-   The **start stage** will run when your container is started. This stage should include any commands needed to start and run your application.

Each stage takes a single command that will be run when the stage is reached in your container build.

```json
//Install stage command for a Node Project
"install_stage": {
    "command": "yarn install"
}
```

### Example: A Rust Devbox

An example of a devbox configuration for a Rust project called `hello_world` might look like the following:

```json
{
    "packages": [
        "rustc"
        "cargo",
        "libiconv",
    ],
    "install_stage": {
        "command": "cargo install --path ."
    },
    "build_stage":{
        "command":"cargo build"
    },
    "start_stage": {
        "command": "./target/build/hello_world"
    }
}
```
