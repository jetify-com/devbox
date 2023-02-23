---
title: Python
---

Python by default will attempt to install your packages globally, or in the Nix Store (which it does not have permissions to modify). To use Python with Devbox, we recommend setting up a Virtual Environment using pipenv or Poetry (see below).

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/development/python)

## Adding Python to your Project

`devbox add python310`, or in your `devbox.json` add:


```json
  "packages": [
    "python310"
  ],
```

This will install Python 3.10 in your shell.

Other versions available include: 

* python37 (Python 3.7)
* python38 (Python 3.8)
* python39 (Python 3.9)
* python311 (Python 3.11)

## Installing Packages with Pip

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/development/python/pip)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/github.com/jetpack-io/devbox-examples?folder=development/python/pip)

[pip](https://pip.pypa.io/en/stable/) is the standard package manager for Python. Since it installs python packages globally, we strongly recommend using a virtual environment.

You can install `pip` by running `devbox add python3xxPackages.pip`, where `3xx` is the version of Python you want to install. This will also install the pip plugin for Devbox, which automatically creates a virtual environment for installing your packages locally

Your virtual environment is created in the `.devbox/virtenv/pip` directory by default, and can be activated by running `source $VENV_DIR/bin/activate` in your devbox shell. You can activate the virtual environment automatically using the init_hook of your `devbox.json`:

```json
{
    "packages": [
        "python310",
        "python310Packages.pip"
    ],
    "shell": {
        "init_hook": ". $VENV_DIR/bin/activate"
    }
}
``` 

## Pipenv

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/development/python/pipenv)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/github.com/jetpack-io/devbox-examples?folder=development/python/pipenv)

[pipenv](https://pipenv.pypa.io/en/latest/) is a tool that will automatically set up a virtual environment for installing your PyPi packages. 

You can install `pipenv` by adding it to the packages in your `devbox.json`. You can then manage your packages and virtual environment via a Pipfile

```json
{
    "packages": [
        "python310",
        "pipenv"
    ],
    "shell": {
        "init_hook": "pipenv shell"
    }
}
```
This init_hook will automatically start your virtualenv when you run `devbox shell`.

## Poetry

[**Example Link**](https://github.com/jetpack-io/devbox-examples/tree/main/development/python/poetry/poetry-demo)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/github.com/jetpack-io/devbox-examples?folder=development/python/poetry/poetry-demo)

[Poetry](https://python-poetry.org/) is a packaging and dependency manager for Python that helps you manage your Python packages, and can automatically create a virtual environment for your project. 

You can install Poetry by adding it to the packages in your `devbox.json`. You can then manage your packages and virtual environment via a `pyproject.toml`

```json
{
    "packages": [
        "python3",
        "poetry"
    ],
    "shell": {
        "init_hook": "poetry shell"
    }
}
```
This init_hook will automatically start Poetry's virtualenv when you run `devbox shell`, and provide you with access to all your packages