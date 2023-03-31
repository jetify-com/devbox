# Contributing

When contributing to this repository, please describe the change you wish to make via a related issue, or a pull request.

Please note we have a [code of conduct](CODE_OF_CONDUCT.md), please follow it in all your interactions with the project.

## Setting Up Development Environment

Before making any changes to the source code (documentation excluded) make sure you have installed all the required tools.

### With Devbox

The easiest way to develop Devbox is with Devbox!

* Install Devbox using the command below. If you don't have Nix installed, Devbox will automatically install it for you when you run a command.

```bash
curl -fsSL https://get.jetpack.io/devbox | bash
```

* You can start up a development shell with all the dependencies installed by running

```bash
devbox shell
```

* Build the Devbox CLI. Note: You can run these commands outside of Devbox Shell

```bash
# Build for your current OS
devbox run build

# Build for Linux
devbox run build-linux

# Run the CI Linter
devbox run lint
```

* Run and Test the Devbox CLI:

```bash
./dist/devbox <test_command>
```

* For the best experience working on Devbox with VSCode, we recommend starting VSCode from inside your Devbox shell. You can also run:

```bash
devbox run code
```

### Setting up the Environment Without Devbox

If you are unable to install or use Devbox, you can manually replicate the environment by following the steps below

* Install [Nix Package Manager](https://nixos.org/download.html).
* Install [Golang](https://go.dev/doc/install) (current version: 1.20)
* Clone this repository:

    ```bash
        git clone github.com/jetpack/devbox go.jetpack.io
    ```

* Setup you `GOPATH` env variable to the parent directory of `go.jetpack.io/`
  * Example: If the cloned repository is at `/Users/johndoe/projects/go.jetpack.io/`:

    ```bash
    export GOPATH=/Users/johndoe/projects/

* Install dependencies:

    ```bash
    go install
    ```

* Build Devbox:

    ```bash
    go build -o ./dist/devbox cmd/devbox/main.go
    ```

    This will build an executable file.

* Run and test Devbox:

    ```bash
    ./dist/devbox <your_test_command>
    ```

## Pull Request Process

1. Ensure any new feature or functionality also includes tests to verify its correctness.

2. Ensure any new dependency is also included in [go.mod](go.mod) file

3. Ensure any binary file as a result of build (e.g., `./devbox`) are removed and/or excluded from tracking in git.

4. Update the [README.md](README.md) and/or docs with details of changes to the interface, this includes new environment
   variables, new commands, new flags, and useful file locations.

5. You may merge the Pull Request in once you have the sign-off of developers/maintainers, or if you
   do not have permission to do that, you may request the maintainers to merge it for you.

## Developer Certificate of Origin

By contributing to this project you agree to the [Developer Certificate of Origin](https://developercertificate.org/) (DCO) which was created by the Linux Foundation and is a simple statement that you, as a contributor, have the legal right to make the contribution. See the DCO description for details below:
> Developer Certificate of Origin
>
> Version 1.1
>
> Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
>
> Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.
>
>
> Developer's Certificate of Origin 1.1
>
> By making a contribution to this project, I certify that:
>
> (a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or
>
> (b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or
>
> (c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.
>
> (d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
