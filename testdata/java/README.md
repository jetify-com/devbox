# Setting Up Test Cases For Java

## Java with Maven
Maven is an all-in-one CI-CD tool for building testing and deploying Java projects. To setup a sample project with Java and Maven in devbox follow the steps below:

1. Create a dummy folder: `dummy/` and call `devbox init` inside it. Then add the nix-pkg: `devbox add jdk` and `devbox add maven`.
    - Replace `jdk` with the version of JDK you want. Get the exact nix-pkg name from `search.nixos.org`.
2. Then do `devbox shell` to get a shell with that `jdk` nix pkg.
3. Then do: `mvn archetype:generate -DgroupId=com.devbox.mavenapp -DartifactId=devbox-maven-app -DarchetypeArtifactId=maven-archetype-quickstart -DarchetypeVersion=1.4 -DinteractiveMode=false`
    - In the generated `pom.xml` file, replace java version in `<maven.compiler.source>` with the specific version you are testing for.
4. `mvn package` should compile the package and create a `target/` directory.
5. `java -cp target/devbox-maven-app-1.0-SNAPSHOT.jar com.devbox.mavenapp.App` should print "Hello World!".
6. Add `target/` to `.gitignore`.