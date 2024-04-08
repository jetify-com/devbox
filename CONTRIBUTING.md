# Contributing

When contributing to this repository, please describe the change you wish to make via a related issue, or a pull request.

Please note we have a [code of conduct](CODE_OF_CONDUCT.md), please follow it in all your interactions with the project.

## Setting Up Development Environment

Before making any changes to the source code (documentation excluded) make sure you have installed all the required tools.

### With Devbox

The easiest way to develop Devbox is with Devbox!

1. Install Devbox:

<<<<<<< HEAD
       curl -fsSL https://get.jetify.com/devbox | bash

2. Clone this repository:

       git clone https://github.com/jetify-com/devbox.git go.jetify.com/devbox
       cd go.jetify.com/devbox
=======
    curl -fsSL https://get.jetify.com/devbox | bash

2. Clone this repository:

    git clone https://github.com/jetify-com/devbox.git go.jetpack.io/devbox
    cd go.jetpack.io/devbox
>>>>>>> 895c1b35 (update github links)

3. Build the Devbox CLI. If you don't have Nix installed, Devbox will automatically install it for you before building:

    devbox run build

4. Start a development shell using your build of Devbox:

    dist/devbox shell

Tip: you can also start VSCode from inside your Devbox shell with `devbox run code`.

-   If you are encountering an error similar to: `line 3: command 'code' not found`, this means you do not have the Visual Studio Code "Shell Command" installed. To do this, follow the official guide: https://code.visualstudio.com/docs/setup/mac. Please refer to the section under: "Launching from the command line".

### Setting up the Environment Without Devbox

If you are unable to install or use Devbox, you can manually replicate the environment by following the steps below.

1.  Install Nix Package Manager. We recommend using the [Determinate Systems installer](https://github.com/DeterminateSystems/nix-installer):

        curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install

    Alternatively, you can also use [the official installer](https://nixos.org/download.html).

2.  Install [Go](https://go.dev/doc/install) (current version: 1.20)

3.  Clone this repository and build Devbox:

<<<<<<< HEAD
       git clone https://github.com/jetify-com/devbox.git go.jetify.com/devbox
       cd go.jetify.com/devbox
       go build ./cmd/devbox
       ./devbox run -- echo hello, world
=======
    git clone https://github.com/jetify-com/devbox.git go.jetpack.io/devbox
    cd go.jetpack.io/devbox
    go build ./cmd/devbox
    ./devbox run -- echo hello, world
>>>>>>> 895c1b35 (update github links)

## Pull Request Process

1. For new features or non-trivial changes, consider first filing an issue to discuss what changes you plan on making. This will let us help you with implementation details and to make sure we don't duplicate any work.
2. Ensure any new feature or functionality includes tests to verify its correctness.
3. Run `devbox run lint` and `devbox run test`.
4. Run `go mod tidy` if you added any new dependencies.
5. Submit your pull request and someone will take a look!

### Style Guide

We don't expect you to read through a long style guide or be an expert in Go before contributing. When necessary, a reviewer will be happy to help out with any suggestions around code style when you submit your PR. Otherwise, the Devbox codebase generally follows common Go idioms and patterns:

-   If you're unfamiliar with idiomatic Go, [Effective Go](https://go.dev/doc/effective_go) and the [Google Go Style Guide](https://google.github.io/styleguide/go) are good resources.
-   There's no strict commit message format, but a good practice is to start the subject with the name of the Go packages you add/modified. For example, `boxcli: update help for add command`.

## Developer Certificate of Origin

By contributing to this project you agree to the [Developer Certificate of Origin](https://developercertificate.org/) (DCO) which was created by the Linux Foundation and is a simple statement that you, as a contributor, have the legal right to make the contribution. See the DCO description for details below:

> By making a contribution to this project, I certify that:
>
> a. The contribution was created in whole or in part by me and I have the right to submit it under the open source license indicated in the file; or
>
> b. The contribution is based upon previous work that, to the best of my knowledge, is covered under an appropriate open source license and I have the right under that license to submit that work with modifications, whether created in whole or in part by me, under the same open source license (unless I am permitted to submit under a different license), as indicated in the file; or
>
> c. The contribution was provided directly to me by some other person who certified (a), (b) or (c) and I have not modified it.
>
> d. I understand and agree that this project and the contribution are public and that a record of the contribution (including all personal information I submit with it, including my sign-off) is maintained indefinitely and may be redistributed consistent with this project or the open source license(s) involved.
