#!/bin/sh
set -eu
STATE_FILE="$DEVBOX_PROJECT_ROOT/.devbox/venv_check_completed"

is_valid_venv() {
    [ -f "$1/bin/activate" ] && [ -f "$1/bin/python" ]
}

is_devbox_venv() {
    [ "$1/bin/python" -ef "$DEVBOX_PACKAGES_DIR/bin/python" ]
}

create_venv() {
    python -m venv "$VENV_DIR" --clear
    echo "*\n.*" >> "$VENV_DIR/.gitignore"
}

# Check that Python version supports venv
if ! python -c 'import venv' 1> /dev/null 2> /dev/null; then
    echo "WARNING: Python version must be > 3.3 to create a virtual environment."
    touch "$STATE_FILE"
    exit 1
fi

# Check if the directory exists
if [ -d "$VENV_DIR" ]; then
    if is_valid_venv "$VENV_DIR"; then
        # Check if we've already run this script
        if [ -f "$STATE_FILE" ]; then
            # "We've already run this script. Exiting..."
            exit 0
        fi
        if ! is_devbox_venv "$VENV_DIR"; then
            echo "WARNING: Virtual environment at $VENV_DIR doesn't use Devbox Python."
            echo "Do you want to overwrite it? (y/n)"
            read reply
            echo
            if [[ $reply =~ ^[Yy]$ ]]; then
                echo "Overwriting existing virtual environment..."
                create_venv
            elif [[ $reply =~ ^[Nn]$ ]]; then
                echo "Using your existing virtual environment. We recommend changing \$VENV_DIR to a different location"
                touch "$STATE_FILE"
                exit 0
            else
                echo "Invalid input. Exiting..."
                exit 1
            fi
        fi
    else
        echo "Directory exists but is not a valid virtual environment. Creating a new one..."
        create_venv
    fi
else
    echo "Virtual environment directory doesn't exist. Creating new one..."
    create_venv
fi
