---
title: Java and Maven
---

### Detection

Devbox will automatically create a Java + Maven Build plan whenever a `pom.xml` is detected in the project's root directory.


### Supported Versions

Devbox will attempt to detect the version set in `<maven.compiler.source>` field of the `pom.xml` file. The following major versions are supported:

- Java 8
- Java 11
- Java 17 (default choice)

If no version is set, Devbox will use Java 17 as the default version.

### Included Nix Packages

- Depending on the detected SDK Version:
    - `jdk8`
    - `jdk11`
    - `jdk17_headless`
    - `jdk` (default choice - Java version 17)
- All other Packages Installed:
    - `maven`
    - `binutils`

### Default Stages

These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details

### Install Stage

```bash
mvn clean install
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

```bash
./customjre/bin/java -jar target/<ArtifactId>-<Version>.jar
```
`<ArtifactId>` and `<Version>` are derived from `pom.xml`