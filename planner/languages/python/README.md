# Python Planner

* We currently support shell and build using poetry and pip. We generally recommend using poetry, but will do our best to support both. 

# Python Poetry Planner

## Detection

* This planner looks for `poetry.lock` or `pyproject.toml` in your devbox.json directory.

## Shell

* poetry by default uses virtual environment so there should be no changes to dev workflow.

## Build

This planner uses pex to build an executable that has all dependencies. It looks for an entrypoint in the following order:

* If there's a module with same name as the project, it uses that as the entrypoint.
* If there's a script with same name as project, it uses that as the entrypoint.
* Use first script in alphabetical order as the entrypoint.

# Python Pip Planner

## Detection

* Looks for `requirements.txt` in your devbox.json directory. We default to python3 as provided by nix packages.

## Shell

* Uses venv (automatically created in .venv) to create a virtual environment.

## Build

* Uses pex to build an executable that has all dependencies. 
* Requires barebones `setup.py`
* Uses `__main__` module in package that matches the lowercase `setup.py` name. Any dashes in the name are replaced with underscores.

# Limitations

Currently `build` does not support projects that depend on libraries with native extensions (e.g. `pandas`).
