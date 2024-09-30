---
title: Devbox Env Variables
---

The following is a list of Environment variables used by Devbox to manage your environment. Some of these variables are set by Devbox, while others can be used to manage how Devbox sets up your shell.

Note that

## Environment Variables Set by Devbox Shell

Below are some useful environment variables that Devbox sets up for you. These variables can be used in your scripts to access information or write scripts for your current project environment.

| Variable | Description |
|:--------|:-----------|
|`DEVBOX_CONFIG_DIR`| The directory where Devbox stores user-editable config files for your project's packages. These files can be checked into source control|
|`DEVBOX_PACKAGES_DIR`| The directory containing the binaries and files for the packages in your current project.|
| `DEVBOX_PROJECT_ROOT` | The root directory of your current project. This will match the directory location of the `devbox.json` file for your currently running shell |
| `DEVBOX_PURE_SHELL` | Indicates whether you are running your current shell in pure mode. Pure mode clears your current environment variables before starting a devbox shell |
| `DEVBOX_SHELL_ENABLED` | Whether or not Devbox is enabled in the current shell. This is automatically set to `1` when you start a shell, run a script, or start services |
| `DEVBOX_WD` | Your current working directory. This can be used when writing scripts that you want to run in your current directory, instead of DEVBOX_PROJECT_ROOT |


## Environment Variables for Managing Devbox

Below are some environment variables that can be used to manage how Devbox sets up your shell. These variables can be set in your shell or in your `devbox.json` file.

| Variable | Description | Value |
|:--------|:-----------|:------------|
|`DEVBOX_DEBUG` | Enable debug output for Devbox. If set to 1, this will print out additional information about what Devbox is doing. | 0 |
|`DEVBOX_FEATURE_DETSYS_INSTALLER` | If enabled, Devbox will use the Determinate Systems installer to setup Nix on your system. | 0 |
|`DEVBOX_NO_PROMPT` | Disables the default shell prompt modification for Devbox. Usually used if you want to configure your own prompt for indicating that you are in a devbox sell | 0 |
|`DEVBOX_PC_PORT_NUM` | Sets the port number for process-compose when running Devbox services. If this variable is unset and a port is not provided via the CLI, Devbox will choose a random available port | `unset` |
|`DEVBOX_USE_VERSION` | Setting this variable will force Devbox to use a different version than the current latest. For example: `DEVBOX_USE_VERSION=0.13.0` will install and use Devbox v0.13 for all Devbox commands| `unset`|
