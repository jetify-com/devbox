# Devbox: instant, easy and predictable development environments.

[![release](https://github.com/jetpack-io/devbox/actions/workflows/release.yml/badge.svg)](https://github.com/jetpack-io/devbox/actions/workflows/release.yml)
![Apache 2.0](https://img.shields.io/github/license/jetpack-io/devbox)

With Devbox, you can easily create deterministic shells with preinstalled utilities without polluting your machine with contradictory versions.

Want to try out a tool but don’t want the mess? Add it to a Devbox shell.

<iframe width="560" height="315" src="https://www.youtube.com/embed/WMBaXQZmDoA" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>

## Compatibility

Devbox works on:

- Linux
- macOS
- Windows via WSL2


## Setup

1. Install [Nix Package Manager](https://nixos.org/download.html). (Don't worry, you don't need to learn Nix.)

2. Install [Docker Engine](https://docs.docker.com/engine/install/) or [Docker Desktop](https://www.docker.com/get-started/).

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

   This creates `devbox.json`. You should commit this file.

3. Add command-line tools from [Nix Packages](https://search.nixos.org/packages). For example, to add Python 3.10:

   ```bash
   devbox add python310
   ```

4. Start a new shell that has these tools installed:

   ```bash
   devbox shell
   ```

   You can tell you’re in a Devbox shell (and not your regular terminal) because the shell prompt and directory changed.

5. Use your favorite tools.

   In this example we installed Python 3.10, so let’s use it.

   ```bash
   python --version
   ```

6. Your regular tools are also available including environment variables and config settings.

   ```bash
   git config --get user.name
   ```

7. To exit the Devbox shell and return to your regular shell:

   ```bash
   exit
   ```


## Quickstart: Automatic Docker Image

With a Devbox environment, it’s simple to build the codebase into an OCI-compliant container image. Devbox will automatically detect your toolchain and pull in the correct Dockerfile.

1. Open a terminal in a Devbox folder (see above).

2. Build the image:

   ```bash
   devbox build
   ```

   The resulting image is named `devbox`.

3. Tag the image with a more descriptive name:

   ```
   docker tag devbox my-image:v0.1
   ```

### Auto-detected languages:

- Go

Want more languages? [Ask for a new Language](https://jetpack-io.canny.io/devbox) or [Contribute one via Pull Request](https://github.com/jetpack-io/devbox/tree/main/tmpl)


## Additional commands

`devbox –help` - see all commands

`devbox plan` - see the configuration and steps Devbox will use to generate a container


## Related Work

Thanks to [Nix](https://nixos.org/) for providing isolated shells. Devbox is not affiliated with the NixOS project.


## License

This project is proudly open-source under the [Apache 2.0 License](https://github.com/jetpack-io/devbox/blob/main/LICENSE) Copyright Jetpack Technologies, Inc.
