#!/bin/sh

poetry env --directory="${DEVBOX_PYPROJECT_DIR:-$DEVBOX_DEFAULT_PYPROJECT_DIR}" --no-interaction --quiet >&2
