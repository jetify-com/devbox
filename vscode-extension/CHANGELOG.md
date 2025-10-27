# Change Log

All notable changes to the "devbox" extension will be documented in this file.

Check [Keep a Changelog](http://keepachangelog.com/) for recommendations on how to structure this file.

## [0.1.7]

- Removed Open In Desktop feature since devbox.sh web app is deprecated.

## [0.1.6]

- Fixed an issue where reopen in devbox feature wasn't working for cursor and vscodium.
- Removed remote-ssh as a dependency extension.

## [0.1.5]

- Rebranding changes from jetpack.io to jetify.com.

## [0.1.4]

- Added debug mode in extension settings (only supports logs for "Reopen in Devbox Shell environment" feature).

## [0.1.3]

- Added json validation for devbox.json files.

## [0.1.2]

- Fixed error handling when using `Reopen in Devbox shell` command in Windows and WSL

## [0.1.1]

- Fixed documentation
- Added devbox install command
- Added devbox update command
- Added devbox search command

## [0.1.0]

- Added reopen in devbox shell environment feature that allows projects with devbox.json
  reopen vscode in devbox environment. Note: It requires devbox CLI v0.5.5 and above
  installed and in PATH. This feature is in beta. Please report any bugs/issues in [Github](https://github.com/jetify-com/devbox) or our [Discord](https://discord.gg/jetify).

## [0.0.7]

- Fixed a bug for `Open in VSCode` that ensures the directory in which
  we save the VM's ssh key does exist.

## [0.0.6]

- Fixed a small bug connecting to a remote environment.
- Added better error handling and messages if connecting to devbox cloud fails.

## [0.0.5]

- Added handling `Open In VSCode` button with `vscode://` style links.
- Added ability for connecting to Devbox Cloud workspace.
- Fixed a bug where devbox extension would run `devbox shell` when opening
a new terminal in vscode even if there was no devbox.json present in the workspace.

## [0.0.4]

- Added `Generate a Dockerfile from devbox.json` to the command palette
- Changed `Generate Dev Containers config files` command's logic to use devbox CLI.

## [0.0.3]

- Small fix for DevContainers and Github CodeSpaces compatibility.

## [0.0.2]

- Added ability to run devbox commands from VSCode command palette
- Added VSCode command to generate DevContainer files to run VSCode in local container or Github CodeSpaces.
- Added customization in settings to turn on/off automatically running `devbox shell` when a terminal window is opened.

## [0.0.1]

- Initial release
- When VScode Terminal is opened on a devbox project, this extension detects `devbox.json` and runs `devbox shell` so terminal is automatically in devbox shell environment.
