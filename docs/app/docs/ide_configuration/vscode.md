---
title: Visual Studio Code 
---


## Java
___
VS Code is a popular editor that supports many different programming languages. This guide covers how to configure VS Code to work with a devbox Java environment.

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

If this guide is missing something, feel free to contribute by opening a [pull request](https://github.com/jetpack-io/devbox/pulls) in Github.