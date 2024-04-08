# Python

Python by default will attempt to install your packages globally, or in the Nix Store (which it does not have permissions to modify). To use Python with Devbox, we recommend setting up a Virtual Environment using pipenv or Poetry (see below).

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/python)

## Adding Python to your Project

`devbox add python@3.10`, or in your `devbox.json` add:

```json
  "packages": [
    "python@3.10"
  ],
```

This will install Python 3.10 in your shell. You can find other versions of Python by running `devbox search python`. You can also view the available versions on [Nixhub](https://www.nixhub.io/packages/python)

## Installing Packages with Pip

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/python/pip)

[![Open In Devbox.sh](https://www.jetify.com/img/devbox/open-in-devbox.svg)](https://devbox.sh/open/templates/python-pip)

[pip](https://pip.pypa.io/en/stable/) is the standard package manager for Python. Since it installs python packages globally, we strongly recommend using a virtual environment.

You can install `pip` by running `devbox add python3xxPackages.pip`, where `3xx` is the version of Python you want to install. This will also install the pip plugin for Devbox, which automatically creates a virtual environment for installing your packages locally

Your virtual environment is created in the `.devbox/virtenv/pip` directory by default, and can be activated by running `. $VENV_DIR/bin/activate` in your devbox shell. You can activate the virtual environment automatically using the init_hook of your `devbox.json`:

```json
{
    "packages": ["python310", "python310Packages.pip"],
    "shell": {
        "init_hook": ". $VENV_DIR/bin/activate"
    }
}
```
