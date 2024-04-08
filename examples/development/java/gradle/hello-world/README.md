# Java

In addition to installing the JDK, you'll need to install either the Maven or Gradle build systems in your shell.

In both cases, you'll want to first activate `devbox shell` before generating your Maven or Gradle projects, so that the tools use the right version of the JDK for creating your project.

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/java)

## Adding the JDK to your project

`devbox add jdk binutils`, or in your `devbox.json`

```json
  "packages": [
    "jdk@latest",
    "binutils@latest"
  ],

```

This will install the latest version of the JDK. To find other installable versions of the JDK, run `devbox search jdk`.

Other distributions of the JDK (such as OracleJDK and Eclipse Temurin) are available in Nixpkgs, and can be found using [NixPkg Search](https://search.nixos.org/packages?channel=22.05&from=0&size=50&sort=relevance&type=packages&query=jdk#)

## Gradle

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/java/gradle/hello-world)

[![Open In Devbox.sh](https://jetify.com/img/devbox/open-in-devbox.svg)](https://devbox.sh/open/templates/java-gradle)

Gradle is a popular, multi-language build tool that is commonly used with JVM projects. To setup an example project using Gradle, follow the instructions below:

1. Create a project folder: `my-project/` and call `devbox init` inside it. Then add these packages: `devbox add jdk` and `devbox add gradle`.
    - Replace `jdk` with the version of JDK you want. Get the exact nix-pkg name from `search.nixos.org`.
2. Then do `devbox shell` to get a shell with that `jdk` nix pkg.
3. Then do: `gradle init`
    - In the generated `build.gradle` file, put the following text block:

        ```gradle
        /* build.gradle */
        apply plugin: 'java'
        apply plugin: 'application'
        /* Change these versions to the JDK version you have installed */
        sourceCompatibility = 17
        targetCompatibility = 17
        mainClassName = 'hello.HelloWorld'
        jar {
            manifest {
              /* assuming main class is in src/main/java/hello/HelloWorld.java */
                attributes 'Main-Class': 'hello.HelloWorld'
            }
        }
        ```

    - While in devbox shell, run `echo $JAVA_HOME` and take note of its value.
    - Create a `gradle.properties` file like below and put value of `$JAVA_HOME` instead of <JAVA_HOME_VALUE> in the file.

      ```gradle
      /* gradle.properties */
      org.gradle.java.home=<JAVA_HOME_VALUE>
      ```

4. `gradle build` should compile the package and create a `build/` directory that contains an executable jar file.
5. `gradle run` should print "Hello World!".
6. Add `build/` to `.gitignore`.

An example `devbox.json` would look like the following:

```json
{
  "packages": [
    "gradle",
    "jdk",
    "binutils"
  ],
  "shell": {
    "init_hook": null
  }
}
```
