---
title: Eclipse integration
---


## Java
This guide describes how to configure Eclipse to work with a devbox Java environment.

### Setting up Devbox shell
To create a devbox shell make sure to have devbox installed. If you don't have devbox installed follow the installation guide first. Then follow the steps below:

1. `devbox init` if you don't have a devbox.json in the root directory of your project.
2. `devbox add jdk` to make sure jdk gets installed in your devbox shell.
3. `devbox shell` to activate and go into your devbox shell.
4. `echo $JAVA_HOME` and copy the path to your java home inside the devbox shell. It should be something like this:
    ```bash
    /nix/store/qaf9fysymdoj19qtyg7209s83lajz65b-zulu17.34.19-ca-jdk-17.0.3
    ```
5. Open Eclipse IDE and create a new Java project if you don't have already
6. From the top menu go to Run > Run Configurations > JRE and choose **Alternate JRE:**
7. Click on **Installed JREs...**  and click **Add...** in the window of Installed JREs.
8. Choose **Standard VM** as JRE Type and click Next.
9. Paste the value you copied in step 4 in **JRE HOME** and put an arbitrary name such as "devbox-jre" in **JRE Name** and click Finish.
10. Click **Apply and Close** in Installed JREs window. Then close Run Configurations.

Now your project in Eclipse is setup to compile and run with the same Java that is installed in your devbox shell. Next step is to run your Java code inside Devbox.

### Setting up Eclipse Terminal

To make sure the compiling and running your Java project happens inside devbox. We have to use a terminal. The following steps show how to run a Java application in Eclipse terminal but the steps are not exclusive to Eclipse and can be used by any unix terminal.

1. Press `ctrl + alt/opt + T` to open terminal window in Eclipse.
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
