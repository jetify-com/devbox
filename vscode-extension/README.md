# devbox VSCode Extension

This is the official VSCode extension for [devbox](https://github.com/jetpack-io/devbox) open source project by [jetpack.io](https://jetpack.io)

## Features

### Open In VSCode button

If a Devbox Cloud instance (from [devbox.sh](https://devbox.sh)) has an `Open In Desktop` button, this extension will make VSCode to be able to connect its workspace to the instance.

### Auto Shell on a devbox project

When VSCode Terminal is opened on a devbox project, this extension detects `devbox.json` and runs `devbox shell` so terminal is automatically in devbox shell environment. Can be turned off in settings.

### Run devbox commands from command palette

`cmd/ctrl + shift + p` opens vscode's command palette. Typing devbox filters all available commands devbox extension can run. Those commands are:

- **Init:** Creates a devbox.json file
- **Add:** adds a package to devbox.json
- **Remove:** Removes a package from devbox.json
- **Shell:** Opens a terminal and runs devbox shell
- **Run:** Runs a script from devbox.json if specified
- **Generate DevContainer files:** Generates devcontainer.json & Dockerfile inside .devcontainers directory. This allows for running vscode in a container or GitHub Codespaces.
- **Generate a Dockerfile from devbox.json:** Generates a Dockerfile a project's root directory. This allows for running the devbox project in a container.

---

## Following extension guidelines

Ensure that you've read through the extensions guidelines and follow the best practices for creating your extension.

- [Extension Guidelines](https://code.visualstudio.com/api/references/extension-guidelines)
