# Python with Poetry Example

Poetry automatically configures a virtual environment for installing your Python packages. This environment can be activated by running `poetry shell` from within your poetry project.

For more information, see the [Poetry Docs](https://python-poetry.org/docs/basic-usage/)

## How to Run

In this directory, run:

`devbox shell`

To activate your poetry shell add `"eval $(poetry env activate)"` to the `init_hook` otherwise use poetry to run commands, e.g. `poetry run pytest`.

To exit the poetry shell, use `exit`. To exit your devbox shell, use `exit` again.

## Configuration

Since Poetry automatically configures a virtual environment for you, no additional Devbox configuration is needed. You can mange your packages and projects.
