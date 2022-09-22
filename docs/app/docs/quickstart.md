---
title: Quickstart
sidebar_position: 3
---


## Create an isolated development shell

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

## Package your application as a Docker Image

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

Want more languages? [Ask for a new Language](https://github.com/jetpack-io/devbox/issues) or contribute one via a Pull Request.