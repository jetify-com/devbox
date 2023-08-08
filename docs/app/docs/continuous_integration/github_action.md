---
title: Using Devbox in CI/CD with Github Actions
---

This guide explains how to use Devbox in CI/CD using Github Actions. The [devbox-install-action](https://github.com/marketplace/actions/devbox-installer) will install Devbox CLI and any packages + configuration defined in your `devbox.json` file. You can then run tasks or scripts within `devbox shell` to reproduce your environment.

This Github Action also supports caching the packages and dependencies installed in your `devbox.json`, which can significantly improve CI build times. 

## Usage

`devbox-install-action` is available on the [Github Marketplace](https://github.com/marketplace/actions/devbox-installer) 

In your project's workflow YAML, add the following step: 

```yaml
- name: Install devbox
  uses: jetpack-io/devbox-install-action@v0.2.0
```

## Example Workflow

The workflow below shows how to use the action to install Devbox, and then run arbitrary commands or [Devbox Scripts](../guides/scripts.md) in your shell.

```yaml
name: Testing with devbox

on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install devbox
        uses: jetpack-io/devbox-install-action@v0.2.0

      - name: Run arbitrary commands
        run: devbox run -- echo "done!"

      - name: Run a script called test
        run: devbox run test
```

## Configuring the Github Action

The `devbox-install-action` provides the following inputs: 

| Input Argument| Description|  Default|
| :- | :- | :- |
|`project-path` | Path to the folder that contains a valid devbox.json	| Root directory of your repo
|`enable-cache` | Caches the entire Nix store (your packages) in Github based on your `devbox.json`.|
|`devbox-version`| Pins a specific version of the Devbox CLI for your action. Only supports >0.2.2| latest|

An example of this configuration is below: 

```yaml
- name: Install devbox
  uses: jetpack-io/devbox-install-action@v0.2.0
  with:
    project-path: 'path-to-folder'
    enable-cache: true
    devbox-version: '0.2.2'
```
