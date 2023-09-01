---
title: devbox.json Reference
sidebar_position: 5
---

Your devbox configuration is stored in a `devbox.json` file, located in your project's root directory. This file can be edited directly, or using the [devbox CLI](cli_reference/devbox.md).

```json
{
    "packages": [] | {},
    "env": {},
    "shell": {
        "init_hook": "...",
        "scripts": {}
    },
    "include": [],
    "nixpkgs": {
        "commit": "..."
    }
}
```

### Packages

This is a list or map of Nix packages that should be installed in your Devbox shell and containers. These packages will only be installed and available within your shell, and will have precedence over any packages installed in your local machine. You can search for Nix packages using [Nix Package Search](https://search.nixos.org/packages).

You can add packages to your devbox.json using `devbox add <package_name>`, and remove them using `devbox rm <package_name>`.

Packages can be structured as a list of package names (`<packages>@<version>`) or [flake references](#adding-packages-from-flakes):

```json
{
    "packages": [
        "go@latest"
        "golangci-lint@latest"
    ]
}
```

If you need to provide more options to your packages (such as limiting which platforms will install the package), you can structure packages as a map, where each package follows the schema below:

```json
{
    "packages": {
        // If only a version is specified, you can abbreviate the maps as "package_name": "version"
        "package_name": string,
        "package_name": {
            // Version of the package to install. Defaults to "latest"
            "version": string,
            // List of platforms to install the package on. Defaults to all platforms
            "platforms": [string],
            // List of platforms to exclude this package from. Defaults to no excluded platforms
            "excluded_platforms": [string]
        }
    }
}
```

For example:

```json
{
    "packages": {
        "go" : "latest",
        "golangci-lint": "latest",
        "glibcLocales": {
            "version": "latest",
            "platforms": ["x86_64-linux, aarch64-linux"]
        }
    }
}
```

Note that `devbox add` will automatically format `packages` based on the options and packages that you provide.

#### Pinning a Specific Version of a Package

You can pin a specific version of a package by adding a `@` followed by the version number to the end of the package name. For example, to pin the `go` package to version `1.19`, you can run `devbox add go@1.19`, or add `go@1.19` to the packages list in your `devbox.json`:

```json
{
    "packages": [
        "go@1.19"
    ]
}
```

Where possible, pinned packages follow semver. For example, if you pin `python@3`, it will install the latest version of `python` with major version `3`.

To see a list of packages and their available versions, you can run `devbox search <pkg>`.

#### Adding Packages from Flakes

You can add packages from flakes by adding a reference to the  flake in the `packages` list in your `devbox.json`. We currently support installing Flakes from Github and local paths.

```json
{
    "packages": [
        // Add the default package from a github repository
        "github:numtide/flake-utils",
        // Install a specific attribute or package from a Github hosted flake
        "github:nix-community/fenix#stable.toolchain",
        // Install a package from a specific channel of Nixpkgs
        "github:nixos/nixpkgs/21.05#hello",
        // Install a package form a specific commit of Nixpkgs
        "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello",
        // Install a package from a local flake. This should point to a directory that contains a flake.nix file.
        "path:../my-flake#my-package"
    ]
}
```

To learn more about using flakes, see the [Using Flakes](guides/using_flakes.md) guide.

#### Adding Platform Specific Packages

You can choose to include or exclude your packages on specific platforms by adding a `platforms` or `excluded_platforms` field to your package definition. This is useful if you need to install packages or libraries that are only available on specific platforms (such as `busybox` on Linux, or `utm` on macOS):

```json
{
    "packages": {
        // Only install busybox on linux
        "busybox": {
            "version": "latest",
            "platforms": ["x86_64-linux", "aarch64-linux"]
        },
        // Exclude UTM on Linux
        "utm": {
            "version": "latest",
            "excluded_platforms": ["x86_64-linux", "aarch64-linux"]
        }
    }
}
```

Note that a package can only specify one of `platforms` or `excluded_platforms`.

Valid Platforms include:

* `aarch64-darwin`
* `aarch64-linux`
* `x86_64-darwin`
* `x86_64-linux`

The platforms below are also supported, but require you to build packages from source:

* `i686-linux`
* `armv7l-linux`

### Env

This is a a map of key-value pairs that should be set as Environment Variables when activating `devbox shell`, running a script with `devbox run`, or starting a service. These variables will only be set in your Devbox shell, and will have precedence over any environment variables set in your local machine or by [Devbox Plugins](guides/plugins.md).

For example, you could set variable `$FOO` to `bar` by adding the following to your `devbox.json`:

```json
{
    "env": {
        "FOO": "bar"
    }
}
```

Currently, you can only set values using string literals, `$PWD`, and `$PATH`. Any other values with environment variables will not be expanded when starting your shell.


### Shell

The Shell object defines init hooks and scripts that can be run with your shell. Right now two fields are supported: *init_hooks*, which run a set of commands every time you start a devbox shell, and *scripts*, which are commands that can be run using `devbox run`

#### Init Hook

The init hook is used to run shell commands before the shell finishes setting up. This hook runs after any other `~/.*rc` scripts, allowing you to override environment variables or further customize the shell.

The init hook will run every time a new shell is started using `devbox shell` or `devbox run`, and is best used for setting up environment variables, aliases, or other quick setup steps needed to configure your environment. For longer running tasks, you should consider using a Script.

This is an example `devbox.json` that customizes the prompt and prints a welcome message:

```json
{
    "shell": {
        "init_hook": [
            "export PS1='📦 devbox> '",
            "echo 'Welcome! See CONTRIBUTING.md for tips on contributing to devbox.'"
        ]
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
{
    "shell": {
        "scripts": {
            "print_once": "echo \"Hello Once!\""
        }
    }
}
```

To run multiple commands in a single script, you can pass them as an array:

```json
{
    "shell": {
        "scripts": {
            "print_twice": [
                "echo \"Hello Once!\"",
                "echo \"Hello Twice!\""
            ]
        }
    }
}
```

### Include

Includes can be used to explicitly add extra configuration or plugins to your Devbox project. Currently this only supports adding our [built-in plugins](guides/plugins.md) to your project.

You should use this section to activate plugins when you install a package from a [Flake](guides/using_flakes.md) that uses a plugin. To ensure that a plugin is activated for your project, add it to the `include` section of your `devbox.json`. For example, to explicitly activate the PHP plugin, you can add the following to your `devbox.json`:

```json
{
    "include": [
        "plugin:php-config"
    ]
}
```

### Nixpkgs

The Nixpkg object is used to optionally configure which version of the Nixpkgs repository you want Devbox to use as the default for installing packages. It currently takes a single field, `commit`, which takes a commit hash for the specific revision of Nixpkgs you want to use.

If a Nixpkg commit is not set, Devbox will automatically add a default commit hash to your `devbox.json`. To upgrade your packages to the latest available versions in the future, you can replace the default hash with the latest nixpkgs-unstable hash from https://status.nixos.org.

To learn more, consult our guide on [setting the Nixpkg commit hash](guides/pinning_packages.md).

### Example: A Rust Devbox

An example of a devbox configuration for a Rust project called `hello_world` might look like the following:

```json
{
    "packages": [
        "rustc",
        "cargo",
        "libiconv"
    ],
    "env": {
        "RUST_BACKTRACE": "1"
    },
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
