---
title: Python
---

## Detection

Devbox will automatically create a Python project plan whenever a `pyproject.toml`, `poetry.lock` or `requirements.txt` file is detected in the project's root directory: 

* If a pyproject.toml or poetry.lock is detected, Devbox will attempt to build your project with Poetry
* If a requirements.txt file is detected, we will attempt to build the project using PIP + setup.py. The build will fail if we cannot find a setup.py in your project.

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

* Depending on the detected Python Version:
  * `python2`
  * `python3`
  * `python37`
  * `python38`
  * `python39`
  * `python310`
  * `python311`
* `poetry`

## Shell Init Hook

When starting a shell for a Python project, Devbox will activate a virtual environment automatically for installing packages.

## Default Stages

These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details.

### Install Stage

#### Poetry Install stage

```bash
poetry add pex -n --no-ansi && poetry install --no-dev -n --no-ansi
```

#### Pip Install Stage

Devbox will first create and activate a virtual environment to install your packages

```bash
python -m venv .venv && source .venv/bin/activate && pip install -r requirements.txt
```

### Build Stage

#### Poetry Build Stage

Devbox will also look for a Poetry script in your Python project to set as your app's entrypoint. If there are multiple scripts configured, Devbox will choose one automatically using the following rules:

1. If Devbox finds a script that matches your module name, it will set that script to run in your Start stage
2. Otherwise, Devbox will run the first script in alphabetical order.

```bash
PEX_ROOT=/tmp/.pex poetry run pex . -o app.pex --script <entrypoint>
```

#### Pip Build Stage

Devbox will first activate your virtual environment, and then run: 

```bash
pip install pex && \
PACKAGE_NAME=$(python setup.py --name |  tr '[:upper:]-' '[:lower:]_') && \
pex . -o app.pex -m $PACKAGE_NAME -r requirements.txt
```

### Start Stage

```bash
python app.pex
```
