---
title: Java
---

In addition to installing the JDK, you'll need to install either the Maven or Gradle build systems in your shell. 

In both cases, you'll want to first activate `devbox shell` before generating your Maven or Gradle projects, so that the tools use the right version of the JDK for creating your project. 

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/development/java)

## Adding the JDK to your project

`devbox add jdk binutils`, or in your `devbox.json`

```json
  "packages": [
    "jdk",
    "binutils"
  ],

```

This will install the OpenJDK version 19. Other versions available include: 

* jdk8 (OpenJDK 8)
* jdk11 (OpenJDK 11)
* jdk17 (OpenJDK 17)

Other distributions of the JDK (such as OracleJDK and Eclipse Temurin) are available in Nixpkgs, and can be found using [NixPkg Search](https://search.nixos.org/packages?channel=22.05&from=0&size=50&sort=relevance&type=packages&query=jdk#)

## Gradle
Gradle is a popular, multi-language build tool that is commonly used with JVM projects. To setup an example project using Gradle, follow the instructions below: 

1. Create a dummy folder: `dummy/` and call `devbox init` inside it. Then add these packages: `devbox add jdk` and `devbox add gradle`.
    - Replace `jdk` with the version of JDK you want. Get the exact nix-pkg name from `search.nixos.org`.
2. Then do `devbox shell` to get a shell with that `jdk` nix pkg.
3. Then do: `gradle init`
    - In the generated `gradle.build` file, put the following text block:
        ```gradle
        apply plugin: 'java'
        apply plugin: 'application'
        sourceCompatibility = 17
        targetCompatibility = 17
        mainClassName = 'hello.HelloWorld'
        jar {
            manifest {
                attributes 'Main-Class': 'hello.HelloWorld'
            }
        }
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

## Maven

Maven is an all-in-one CI-CD tool for building testing and deploying Java projects. To setup a sample project with Java and Maven in devbox follow the steps below:

1. Create a dummy folder: `dummy/` and call `devbox init` inside it. Then add the nix-pkg: `devbox add jdk` and `devbox add maven`.
    - Replace `jdk` with the version of JDK you want. Get the exact nix-pkg name from `search.nixos.org`.
2. Then do `devbox shell` to get a shell with that `jdk` nix pkg.
3. Then do: `mvn archetype:generate -DgroupId=com.devbox.mavenapp -DartifactId=devbox-maven-app -DarchetypeArtifactId=maven-archetype-quickstart -DarchetypeVersion=1.4 -DinteractiveMode=false`
    - In the generated `pom.xml` file, replace java version in `<maven.compiler.source>` with the specific version you are testing for.
4. `mvn package` should compile the package and create a `target/` directory.
5. `java -cp target/devbox-maven-app-1.0-SNAPSHOT.jar com.devbox.mavenapp.App` should print "Hello World!".
6. Add `target/` to `.gitignore`.

An example `devbox.json` would look like the following:
```json
{
  "packages": [
    "maven",
    "jdk",
    "binutils"
  ],
  "shell": {
    "init_hook": null
  }
}
```