SOURCE_DATE_EPOCH=$(date +%s)

if [ -d "$VENV_DIR" ]; then
    echo "Skipping venv creation, '${VENV_DIR}' already exists"
else
    echo "Creating new venv environment in path: '${VENV_DIR}'"
    # Note that the module venv was only introduced in python 3, so for 2.7
    # this needs to be replaced with a call to virtualenv
    which python3
    python3 -m venv "$VENV_DIR"
fi
