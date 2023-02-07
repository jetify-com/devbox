#!/bin/sh

if [ -d "$VENV_DIR" ]; then
    echo "Skipping venv creation, '${VENV_DIR}' already exists"
else
    echo "Creating new venv environment in path: '${VENV_DIR}'"
    python3 -m venv "$VENV_DIR"
fi
echo "You an activate the virtual environment by running 'source \$VENV_DIR/bin/activate'"
