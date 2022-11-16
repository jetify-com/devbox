---
title: Configuration Guide
sidebar_position: 5
---

Your devbox configuration is stored in a `devbox.json` file, located in your project's root directory. This file can be edited directly, or using the [devbox CLI](cli_reference/devbox.md).

```json
{
    "packages": [],
    "shell": {
        "init_hook": "...",
        "scripts": {}
    },
    "nixpkgs": {
        "commit": "..."
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

The Shell object defines init hooks and scripts that can be run with your shell. Right now two fields are supported: *init_hooks*, which run a set of commands every time you start a devbox shell, and *scripts*, which are commands that can be run using `devbox run`

#### Init Hook

The init hook is used to run shell commands before the shell finishes setting up. This hook runs after any other `~/.*rc` scripts, allowing you to override environment variables or further customize the shell. 

The init hook will run every time a new shell is started using `devbox shell` or `devbox run`, and is best used for setting up environment variables, aliases, or other quick setup steps needed to configure your environment. For longer running tasks, you should consider using a Script. 

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

#### Scripts

Scripts are commands that are executed in your Devbox shell using `devbox run <script_name>`. They can be used to start up background process (like databases or servers), or to run one off commands (like setting up a dev DB, or running your tests). 

Scripts can be defined by giving a name, and one or more commands. Single command scripts can be added by providing a name, and a string:

```json
"scripts": {
    "print_once": "echo \"Hello Once!\""
}
```

To run multiple commands in a single script, you can pass them as an array: 

```json
"scripts": {
    "print_twice": [
        "echo \"Hello Once!\"",
        "echo \"Hello Twice!\""
    ]
}
```

### Nixpkgs

The Nixpkg object is used to optionally configure which version of the Nixpkgs repository you want Devbox to use for installing packages. It currently takes a single field, `commit`, which takes a commit hash for the specific revision of Nixpkgs you want to use.

If a Nixpkg commit is not set, Devbox will automatically add a default commit hash to your `devbox.json`. To upgrade your packages to the latest available versions in the future, you can replace the default hash with the latest nixpkgs-unstable hash from https://status.nixos.org

To learn more, consult our guide on [setting the Nixpkg commit hash](guides/pinning_packages.md). 


### Stages

Stages are used to configure and run commands at different points of container creation. For languages that support autodetection, Devbox will automatically detect and configure the correct stage commands for your project based on your source code. You can override any of these stages by configuring them in your devbox.json

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
        "rustc",
        "cargo",
        "libiconv"
    ],
    "shell": {
        "init_hook": [
            "source conf/set-environment.sh",
            "rustup default stable",
            "cargo fetch"
        ],
        "scripts": {
            "test": "cargo test -- --show-output",
            "start" : "cargo run",
            "build-docs": "cargo doc"
        }
    }
}
```
