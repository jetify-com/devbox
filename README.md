# Devbox 📦

### Instant, easy, and predictable shells and containers

[![Join Discord](https://img.shields.io/discord/903306922852245526?color=7389D8&label=discord&logo=discord&logoColor=ffffff)](https://discord.gg/agbskCJXk2) ![License: Apache 2.0](https://img.shields.io/github/license/jetpack-io/devbox) [![version](https://img.shields.io/github/v/release/jetpack-io/devbox?color=green&label=version&sort=semver)](https://github.com/jetpack-io/devbox/releases) [![tests](https://github.com/jetpack-io/devbox/actions/workflows/release.yml/badge.svg)](https://github.com/jetpack-io/devbox/actions/workflows/release.yml)

---

## What is it?

Devbox is a command-line tool that lets you easily create isolated shells and containers. You start by defining the list of packages required by your development environment, and devbox uses that definition to create an isolated environment just for your application.

In practice, Devbox works similar to a package manager like `yarn` – except the packages it manages are at the operating-system level (the sort of thing you would normally install with `brew` or `apt-get`).

Devbox was originally developed by [jetpack.io](https://www.jetpack.io) and is internally powered by `nix`.

## Demo
The example below creates a development environment with `python 2.7` and `go 1.18`, even though those packages are not installed in the underlying machine:

![screen cast](https://user-images.githubusercontent.com/279789/186491771-6b910175-18ec-4c65-92b0-ed1a91bb15ed.svg)


## Benefits

### A consistent shell for everyone on the team

Declare the list of tools needed by your project via a `devbox.json` file and run `devbox shell`. Everyone working on the project gets a shell environment with the exact same version of those tools.

### Try new tools without polluting your laptop

Development environments created by Devbox are isolated from everything else in your laptop. Is there a tool you want to try without making a mess? Add it to a Devbox shell, and remove it when you don't want it anymore – all while keeping your laptop pristine.

### Don't sacrifice speed

Devbox can create isolated environments right on your laptop, without an extra-layer of virtualization slowing your file system or every command. When you're ready to ship, it'll turn it into an equivalent container – but not before.

### Good-bye conflicting versions

Are you working on multiple projects, all of which need different versions of the same binary? Instead of attempting to install conflicting versions of the same binary on your laptop, create an isolated environment for each project, and use whatever version you want for each.

### Instantly turn your application into a container

Devbox analyzes your source code and instantly turns it into an OCI-compliant image that can be deployed to any cloud. The image is optimized for speed, size, security and caching ... and without needing to write a `Dockerfile`. And unlike [buildpacks](https://buildpacks.io/), it does it quickly.

### Stop declaring dependencies twice

Your application often needs the same set of dependencies when you are developing on your laptop, and when you're packaging it as a container ready to deploy to the cloud. Devbox's dev environments are _isomorphic_: meaning that we can turn them into both a local shell environment or a cloud-ready container, all without having to repeat yourself twice.

## Installing Devbox

In addition to installing Devbox itself, you will need to install `nix` and `docker` since Devbox depends on them:

1. Install [Nix Package Manager](https://nixos.org/download.html). (Don't worry, you don't need to learn Nix.)

2. Install [Docker Engine](https://docs.docker.com/engine/install/) or [Docker Desktop](https://www.docker.com/get-started/). Note that docker is only needed if you want to create containers – the shell functionality works without it.

3. Install Devbox:

   ```sh
   curl -fsSL https://get.jetpack.io/devbox | bash
   ```

## Quickstart: Fast, Deterministic Shell

In this quickstart we’ll create a development shell with specific tools installed. These tools will only be available when using this Devbox shell, ensuring we don’t pollute your machine.

1. Open a terminal in a new empty folder.

2. Initialize Devbox:

   ```bash
   devbox init
   ```

   This creates a `devbox.json` file in the current directory. You should commit it to source control.

3. Add command-line tools from [Nix Packages](https://search.nixos.org/packages). For example, to add Python 3.10:

   ```bash
   devbox add python310
   ```
4. Your `devbox.json` file keeps track of the packages you've added, it should now look like this:

   ```json
   {
      "packages": [
         "python310"
       ]
   }
   ```

5. Start a new shell that has these tools installed:

   ```bash
   devbox shell
   ```

   You can tell you’re in a Devbox shell (and not your regular terminal) because the shell prompt and directory changed.

6. Use your favorite tools.

   In this example we installed Python 3.10, so let’s use it.

   ```bash
   python --version
   ```

7. Your regular tools are also available including environment variables and config settings.

   ```bash
   git config --get user.name
   ```

8. To exit the Devbox shell and return to your regular shell:

   ```bash
   exit
   ```

## Quickstart: Instant Docker Image

Devbox makes it easy to package your application into an OCI-compliant container image. Devbox analyzes your code, automatically identifies the right toolchain needed by your project, and builds it into a docker image.

1. Initialize your project with `devbox init` if you haven't already.

2. Build the image:

   ```bash
   devbox build
   ```

   The resulting image is named `devbox`.

3. Tag the image with a more descriptive name:

   ```bash
   docker tag devbox my-image:v0.1
   ```
### Auto-detected languages:
Devbox currently detects the following languages:

- Go
- Python (Poetry)

Want more languages? [Ask for a new Language](https://github.com/jetpack-io/devbox/issues) or contribute one via a Pull Request.

## Additional commands

`devbox help` - see all commands

`devbox plan` - see the configuration and steps Devbox will use to generate a container

## Join our Developer Community

+ Chat with us by joining the [Jetpack.io Discord Server](https://discord.gg/agbskCJXk2) – we have a #devbox channel dedicated to this project. 
+ File bug reports and feature requests using [Github Issues](https://github.com/jetpack-io/devbox/issues)
+ Follow us on [Jetpack's Twitter](https://twitter.com/jetpack_io) for product updates

## Related Work

Thanks to [Nix](https://nixos.org/) for providing isolated shells.

## License

This project is proudly open-source under the [Apache 2.0 License](https://github.com/jetpack-io/devbox/blob/main/LICENSE)
