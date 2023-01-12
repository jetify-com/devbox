---
title: direnv 
---


## direnv
___
[direnv](https://direnv.net) is an open source environment management tool that allows setting unique environment variables per directory in your file system. This guide covers how to configure direnv to seamlessly work with a devbox project.

### Prerequisites
* Install direnv and hook it to your shell. Follow [this guide](https://direnv.net/#basic-installation) if you haven't done it. 

### Setting up Devbox Shell and direnv

#### New Project

If you have direnv installed, Devbox will generate an .envrc file when you run `devbox init` and prompt you to enable it:

```bash
➜  devbox init
? Do you want to enable direnv integration for this devbox project?[y/n] y
direnv: loading ~/src/devbox/docs/.envrc
direnv: using devbox
```

This will generate a `.envrc` file in your root directory along with your `devbox.json`, and run `direnv allow` so that your shell will activate whenever you navigate to the directory.

If you choose not to enable the integration, you can enable it at anytime by running `direnv allow`, or following the global settings below

#### Existing Project

For an existing project, you can add a `.envrc` file by running `devbox generate envrc`:

```bash
➜  devbox generate direnv
? Do you want to enable direnv integration for this devbox project?[y/n] y
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
<!-- TODO: add steps for vscode integration -->

If this guide is missing something, feel free to contribute by opening a [pull request](https://github.com/jetpack-io/devbox/pulls) in Github.