---
title: Python
---

## Detection

Devbox will automatically create a Python project plan whenever a `pyproject.toml` file is detected in the project's root directory. 

## Supported Versions

Devbox will attempt to attempt to detect the version of Python to install using `tool.poetry.dependencies` section of your `pyproject.toml`. The following versions are supported: 

* 2.7
* 3.7
* 3.8
* 3.9
* 3.10
* 3.11

If no version is specified, Devbox will default to Python 3.10
## Included Nix Packages

* Depending on the detected Node Version:
  * `python2`
  * `python37`
  * `python38`
  * `python39`
  * `python310`
  * `python311`
* `poetry`

## Default Stages

These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details

### Install Stage
```bash
poetry add pex -n --no-ansi && poetry install --no-dev -n --no-ansi
```

### Build Stage

Devbox will also look for a Poetry script in your Python project to set as your app's entrypoint. If there are multiple scripts configured, Devbox will choose one automatically using the following rules:

1. If Devbox finds a script that matches your module name, it will set that script to run in your Start stage
2. Otherwise, Devbox will run the first script in alphabetical order.

```bash
PEX_ROOT=/tmp/.pex poetry run pex . -o app.pex --script <entrypoint>
```

### Start Stage

```bash
python /temp/.pex/app.pex
```