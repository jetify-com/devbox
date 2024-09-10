set -eu
STATE_FILE="$DEVBOX_PROJECT_ROOT/.devbox/venv_check_completed"
echo $STATE_FILE

is_valid_venv() {
    [ -f "$1/bin/activate" ] && [ -f "$1/bin/python" ]
}

# Function to check if Python is a symlink to a Devbox Python
is_devbox_python() {
    if [ -z "$DEVBOX_PACKAGES_DIR" ]; then
        echo "DEVBOX_PACKAGES_DIR is not set. Unable to check for Devbox Python."
        return 1
    fi
    local python_path="$1/bin/python"
    local link_target

    while true; do
        if [ ! -L "$python_path" ]; then
            # Not a symlink, we're done
            break
        fi

        link_target=$(readlink "$python_path")
        echo "Checking symlink: $link_target"

        if [[ "$link_target" == /* ]]; then
            # Absolute path, we're done
            python_path="$link_target"
            break
        elif [[ "$link_target" == python* ]] || [[ "$link_target" == ./* ]] || [[ "$link_target" == ../* ]]; then
            # Relative path or python symlink, continue resolving
            python_path=$(dirname "$python_path")/"$link_target"
        else
            # Unexpected format, stop here
            break
        fi
    done

    [[ $python_path == $DEVBOX_PACKAGES_DIR/* ]]
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
    # "We've already run this script. Exiting..."
    exit 0
fi

# Check Python version
if ! check_python_version; then
    echo "\033[1;33mWARNING: Python version must be > 3.3 to create a virtual environment.\033[0m"
    touch "$STATE_FILE"
    exit 1
fi

# Check if the directory exists
if [ -d "$VENV_DIR" ]; then
    if is_valid_venv "$VENV_DIR"; then
        if ! is_devbox_python "$VENV_DIR"; then
            echo "\033[1;33mWARNING: Virtual environment at $VENV_DIR doesn't use Devbox Python.\033[0m"
            echo "Virtual environment: $VENV_DIR"
            read -p "Do you want to overwrite it? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                echo "Overwriting existing virtual environment..."
                rm -rf "$VENV_DIR"
                python3 -m venv "$VENV_DIR"
            else
                echo "Using existing virtual environment. We recommend changing \$VENV_DIR"
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
