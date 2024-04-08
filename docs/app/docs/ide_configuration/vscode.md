---
title: Visual Studio Code 
---

## Devbox Extension
___

Devbox has an accompanying [VSCode extension](vscode:extension/jetify-com.devbox) that makes the experience of integrating your devbox environment in VSCode much simpler. 

### Syncing VSCode with Devbox shell
Follow the steps below to have VSCode's environment be in sync with Devbox shell:

1. [Install](vscode:extension/jetify-com.devbox) Devbox's VSCode extension
2. Open a project that has a devbox.json file in VSCode
3. Open command palette in VSCode (cmd+shift+p) and type: `Devbox: Reopen in Devbox shell environment`
4. Press Enter and wait for VSCode to reload.
5. The newly opened VSCode is now integrated with the environment defined your devbox.json. You can test it by checking if packages defined in devbox.json are available in VSCode integrated terminal.

Keep in mind that if you make changes to your devbox.json, you need to re-run Step 3 to make VSCode pick up the new changes.

**NOTE:** This integration feature requires Devbox CLI v0.5.5 and above installed and in PATH. This feature is in beta. Please report any bugs/issues in [Github](https://github.com/jetify-com/devbox) or our [Discord](https://discord.gg/Rr5KPJq7).

**NOTE2:** This feature is not yet available for Windows and WSL.

### Automatic Devbox shell in VSCode Terminal

Devbox extension runs `devbox shell` automatically every time VSCode's integrated terminal is opened, **if the workspace opened in VSCode has a devbox.json file**. 

This setting can be turned off in VSCode's settings. Simply search for `devbox.autoShellOnTerminal` in settings or add the following to VSCode's settings.json:
```json
"devbox.autoShellOnTerminal": false
```
Note that running `devbox shell` is not necessary if VSCode is reopened in Devbox shell environment via the steps described in [Syncing VSCode with Devbox shell](#syncing-vscode-with-devbox-shell)

## Direnv Extension
___
Direnv is an open source environment management tool that allows setting unique environment variables per directory in your file system. For more details on how to set it and integrate it with Devbox visit [our Direnv setup guide](../direnv/).

Once Direnv is installed and setup with Devbox, its [VSCode extension](vscode:extension/mkhl.direnv) can also be used to integrate the environment defined in your devbox.json to VSCode. To do that follow the steps below:

1. Install Direnv ([link to guide](https://direnv.net/#basic-installation))
2. Setup Devbox shell with Direnv ([link to guide](../direnv/#setting-up-devbox-shell-and-direnv))
3. Install Direnv's [VSCode extension](vscode:extension/mkhl.direnv)
4. Open your Devbox project in VSCode. Direnv extension should show a prompt notification to reload your environment.
5. Click on reload.

## Windows Setup
___
Devbox CLI is not supported on Windows, but you can still use it with VSCode by using Windows Subsystem for Linux ([WSL](https://learn.microsoft.com/en-us/windows/wsl/install)). If you've set up WSL, follow these steps to integrate your Devbox shell environment with VSCode:

1. [Install](https://www.jetify.com/devbox/docs/installing_devbox/) Devbox in WSL.
2. Navigate to your project directory. (`C:\Users` is `/mnt/c/Users/` in WSL).
3. Run `devbox init` if you don't have a devbox.json file.
4. Run `devbox shell`
5. Run `code .` to open VSCode in Windows and connect it remotely to your Devbox shell in WSL.
## Manual Setup
___
VS Code is a popular editor that supports many different programming languages. This section covers how to configure VS Code to work with a devbox Java environment as an example.

### Setting up Run and Debugger
To create a devbox shell make sure to have devbox installed. If you don't have devbox installed follow the installation guide first. Then follow the steps below:

1. `devbox init` if you don't have a devbox.json in the root directory of your project.
2. `devbox add jdk` to make sure jdk gets installed in your devbox shell.
3. `devbox shell -- 'which java` to activate devbox shell temporarily and find the path to your executable java binary inside the devbox shell. Copy and save that path. It should look something like this:
    ```bash
    /nix/store/qaf9fysymdoj19qtyg7209s83lajz65b-zulu17.34.19-ca-jdk-17.0.3/bin/java
    ```
4. Open VS Code and create a new Java project if you don't have already. If VS Code prompts for installing Java support choose yes.
5. Click on **Run and Debug** icon from the left sidebar.
6. Click on **create a launch.json** link in the opened sidebar. If you don't see such a link, click on the small gear icon on the top of the open sidebar.
7. Once the `launch.json` file is opened, update the `configurations` parameter to look like snippet below:
    ```json
    {
        "type": "java",
        "name": "Launch Current File",
        "request": "launch",
        "mainClass": "<project_directory_name>/<main_package>.<main_class>",
        "projectName": "<project_name>",
        "javaExec": "<path_to_java_executable_from_step_4>"
    }
    ```
    Update the values in between < and > to match your project and environment.
8. Click on **Run and Debug** or the green triangle at the top of the left sidebar to run and debug your project.

Now your project in VS Code is setup to run and debug with the same Java that is installed in your devbox shell. Next step is to run your Java code inside Devbox.

### Setting up Terminal

The following steps show how to run a Java application in a devbox shell using the VS Code terminal. Note that most of these steps are not exclusive to VS Code and can also be used in any Linux or macOS terminal.

1. Open VS Code terminal (`ctrl + shift + ~` in MacOS)
2. Navigate to the projects root directory using `cd` command.
3. Make sure `devbox.json` is present in the root directory `ls | grep devbox.json`
4. Run `devbox shell` to activate devbox shell in the terminal.
5. Use `javac` command to compile your Java project. As an example, if you have a simple hello world project and the directory structure such as: 
    ```bash
    my_java_project/
    -- src/
    -- -- main/
    -- -- -- hello.java
    ```
    You can use the following command to compile:
    to compile:
    ```bash
    javac my_java_project/src/main/hello.java
    ```
6. Use `java` command to run the compiled proect. For example, to run the sample project from above:
    ```bash
    cd src/
    java main/hello
    ```

If this guide is missing something, feel free to contribute by opening a [pull request](https://github.com/jetify-com/devbox/pulls) in Github.