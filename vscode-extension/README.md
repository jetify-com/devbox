# devbox VSCode Extension

This is the official VSCode extension for [devbox](https://github.com/jetify-com/devbox) open source project by [jetify.com](https://www.jetify.com)

## Features

### Open In VSCode button

If a Devbox Cloud instance (from [devbox.sh](https://devbox.sh)) has an `Open In Desktop` button, this extension will make VSCode to be able to connect its workspace to the instance.

### Auto Shell on a devbox project

When VSCode Terminal is opened on a devbox project, this extension detects `devbox.json` and runs `devbox shell` so terminal is automatically in devbox shell environment. Can be turned off in settings.

### Reopen in Devbox shell environment

If the opened workspace in VSCode has a devbox.json file, from command palette, invoking the devbox command `Reopen in Devbox shell environment` will do the following:

1. Installs devbox packages if missing.
2. Update workspace settings for MacOS to create terminals without creating a login shell [learn more](https://code.visualstudio.com/docs/terminal/profiles#_why-are-there-duplicate-paths-in-the-terminals-path-environment-variable-andor-why-are-they-reversed-on-macos)
3. Interact with Devbox CLI to setup a devbox shell.
4. Close current VSCode window and reopen it in a devbox shell environment as if VSCode was opened from a devbox shell terminal.

NOTE: Requires devbox CLI v0.5.5 and above
  installed and in PATH. This feature is in beta. Please report any bugs/issues in [Github](https://github.com/jetify-com/devbox) or our [Discord](https://discord.gg/jetify).

### Run devbox commands from command palette

`cmd/ctrl + shift + p` opens vscode's command palette. Typing devbox filters all available commands devbox extension can run. Those commands are:

- **Init:** Creates a devbox.json file
- **Add:** adds a package to devbox.json
- **Remove:** Removes a package from devbox.json
- **Shell:** Opens a terminal and runs devbox shell
- **Run:** Runs a script from devbox.json if specified
- **Install** Install packages specified in devbox.json
- **Update** Update packages specified in devbox.json
- **Search** Search for packages to add to your devbox project
- **Generate DevContainer files:** Generates devcontainer.json & Dockerfile inside .devcontainers directory. This allows for running vscode in a container or GitHub Codespaces.
- **Generate a Dockerfile from devbox.json:** Generates a Dockerfile a project's root directory. This allows for running the devbox project in a container.
- **Reopen in Devbox shell environment:** Allows projects with devbox.json
  reopen VSCode in devbox environment. Note: It requires devbox CLI v0.5.5 and above
  installed and in PATH.

### JSON validation when writing a devbox.json file

No need to take any action for this feature. When writing a devbox.json, if this extension is installed, it will validate and highlight any disallowed fields or values on a devbox.json file.

---

### Debug Mode

Enabling debug mode in extension settings will create a seqience of logs in the file `.devbox/extension.log`. This feature only tracks the logs for `"Devbox: Reopen in Devbox Shell environment"` feature.

## Following extension guidelines

Ensure that you've read through the extensions guidelines and follow the best practices for creating your extension.

- [Extension Guidelines](https://code.visualstudio.com/api/references/extension-guidelines)

## Publishing

Steps:

1. Bump the version in `package.json`, and add notes to `CHANGELOG.md`. Sample PR: #951.
2. Manually trigger the [`vscode-ext-release` in Github Actions](https://github.com/jetify-com/devbox/actions/workflows/vscode-ext-release.yaml).
