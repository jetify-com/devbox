---
title: Java
---

### Detection

#### Maven
Devbox will automatically create a Java + Maven Build plan whenever a `pom.xml` is detected in the project's root directory.

#### Gradle
Devbox will automatically create a Java + Gradle Build plan whenever a `build.gradle` is detected in the project's root directory.

### Supported Versions

Devbox will attempt to detect the version set in depending on the build tool.
If no version is set, Devbox will use Java 17 as the default version.
#### In Maven
`<maven.compiler.source>` field of the `pom.xml` file is used. The following major versions are supported:

- Java 8
- Java 11
- Java 17 (default choice)
#### In Gradle
`sourceCompatibility` field of the `build.gradle` file is used. The following major versions are supported:

- Java 17

Note: Java versions 11, 8 and other versions may work too but are not tested.


### Included Nix Packages

- Depending on the detected SDK Version:
    - `jdk8`
    - `jdk11`
    - `jdk17_headless`
    - `jdk` (default choice - Java version 17)
- All other Packages Installed:
    - `maven` or `gradle`
    - `binutils`

### Default Stages

These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details

### Install Stage

#### For Maven
```bash
mvn clean install
```
#### For Gradle
Devbox planner assumes an executable Gradle wrapper (gradlew) exists in the root directory of the project.
```bash
./gradlew build
```
### Build Stage

```bash
jlink --verbose \
    --add-modules ALL-MODULE-PATH \
    --strip-debug \
    --no-man-pages \
    --no-header-files \
    --compress=2 \
    --output ./customjre
```

### Start Stage

For Maven:
```bash
./customjre/bin/java -jar target/<ArtifactId>-<Version>.jar
```
`<ArtifactId>` and `<Version>` are derived from `pom.xml`

For Gradle:
```bash
export JAVA_HOME=./customjre && ./gradlew run
```
