---
title: direnv configuration
---


## direnv
___
[direnv](https://direnv.net) is an open source environment management tool that allows setting unique environment variables per directory in your file system. This guide covers how to configure direnv to seemlessly work with a devbox project.

### Prerequisites
* Install direnv and hook it to your shell. Follow [this guide](https://direnv.net/#basic-installation) if you haven't done it. 

### Setting up with Devbox Shell and direnv

Note: If you already have a devbox project you may skip to step 3.

1. `devbox init` if you don't have a devbox.json in the root directory of your project.
2. `devbox shell -- 'ls'` to activate devbox shell temporarily and make sure dependencies mentioned in your devbox.json are installed.
3. Create a new file, name it `.envrc` and put the following snippet inside it:
    ```bash
    use_devbox() {
        watch_file devbox.json
        eval $(devbox shell --print-env)
    }
    use devbox
    ```
4. Run `direnv allow` to give permission to `direnv` to setup your environment variables.
5. At this point, your project directory is setup so that every time you `cd` into it, the binaries from your devbox shell will be used. To test this, you can compare running `which python3` from your project directory and outside.

### Global settings for direnv

Note that every time changes are made to `devbox.json` via `devbox add ...`, `devbox rm ...` or directly editing the file, requires `direnv allow` to run so that `direnv` can setup the new changes.

Alternatively, a project directory can be whitelisted so that changes will be automatically picked up by `direnv`. This is done by adding following snippet to direnv config file typically at `~/.config/direnv/direnv.toml`. You can create the file and directory if it doesn't exist.

```toml
[whitelist]
prefix = [ "/absolute/path/to/project" ]

```
<!-- TODO: add steps for vscode integration -->

If this guide is missing something, feel free to contribute by opening a [pull request](https://github.com/jetpack-io/devbox/pulls) in Github.