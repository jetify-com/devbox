STATE_FILE="$DEVBOX_PROJECT_ROOT/.devbox/venv_check_completed"

is_valid_venv() {
    [ -f "$1/bin/activate" ] && [ -f "$1/bin/python" ]
}

# Function to check if Python is a symlink to a Devbox Python
is_devbox_python() {
    if [ -z "$DEVBOX_PACKAGES_DIR" ]; then
        echo "DEVBOX_PACKAGES_DIR is not set. Unable to check for Devbox Python."
        return 1
    fi
    local python_path=$(readlink "$1/bin/python")
    echo $python_path
    echo $DEVBOX_PACKAGES_DIR
    [[ $python_path == $DEVBOX_PACKAGES_DIR/bin/python* ]]
}

# Function to check Python version
check_python_version() {
    python_version=$(python -c 'import sys; print(".".join(map(str, sys.version_info[:2])))')
    if [ "$(printf '%s\n' "3.3" "$python_version" | sort -V | head -n1)" = "3.3" ]; then
        return 0
    else
        return 1
    fi
}

# Check if we've already run this script
if [ -f "$STATE_FILE" ]; then
    exit 0
fi

# Check Python version
if ! check_python_version; then
    echo "\n\033[1;33m========================================\033[0m"
    echo "\033[1;33mWARNING: Python version must be > 3.3 to create a virtual environment.\033[0m"
    echo "\033[1;33m========================================\033[0m"
    touch "$STATE_FILE"
    exit 1
fi

# Check if the directory exists
if [ -d "$VENV_DIR" ]; then
    if is_valid_venv "$VENV_DIR"; then
        if ! is_devbox_python "$VENV_DIR"; then
            echo "\n\033[1;33m========================================\033[0m"
            echo "\033[1;33mWARNING: Existing virtual environment doesn't use Devbox Python.\033[0m"
            echo "\033[1;33m========================================\033[0m"
            echo "Virtual environment: $VENV_DIR"
            read -p "Do you want to overwrite it? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                echo "Overwriting existing virtual environment..."
                rm -rf "$VENV_DIR"
                python3 -m venv "$VENV_DIR"
            else
                echo "Operation cancelled."
                touch "$STATE_FILE"
                exit 1
            fi
        fi
    else
        echo "Directory exists but is not a valid virtual environment. Creating new one..."
        rm -rf "$VENV_DIR"
        python -m venv "$VENV_DIR"
    fi
else
    echo "Virtual environment directory doesn't exist. Creating new one..."
    python -m venv "$VENV_DIR"
fi
