# Contributing

When contributing to this repository, please describe the change you wish to
make via a related issue, or a pull request.

Please note we have a [code of conduct](CODE_OF_CONDUCT.md), please follow it in
all your interactions with the project.

## Setting Up Development Environment

Before making any changes to the source code (documentation excluded) make sure
you have installed all the required tools.

### With Devbox

The easiest way to develop Devbox is with Devbox!

1. Install Devbox:

       curl -fsSL https://get.jetify.com/devbox | bash

2. Clone this repository:

       git clone https://github.com/jetify-com/devbox.git go.jetify.com/devbox
       cd go.jetify.com/devbox

3. Build the Devbox CLI. If you don't have Nix installed, Devbox will
   automatically install it for you before building:

       devbox run build

4. Start a development shell using your build of Devbox:

       dist/devbox shell

Tip: you can also start VSCode from inside your Devbox shell with
`devbox run code`.

- If you encounter an error similar to: `line 3: command 'code' not found`, it
  means you do not have the Visual Studio Code "Shell Command" installed. Follow
  the official guide at https://code.visualstudio.com/docs/setup/mac. Please
  refer to the section under: "Launching from the command line".

### Setting up the Environment Without Devbox

If you are unable to install or use Devbox, you can manually replicate the
environment by following the steps below.

1. Install Nix Package Manager. We recommend using the
   [Determinate Systems installer](https://github.com/DeterminateSystems/nix-installer):

       curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install

   Alternatively, you can also use
   [the official installer](https://nixos.org/download.html).

2. Install [Go](https://go.dev/doc/install) (current version: 1.20)

3. Clone this repository and build Devbox:

       git clone https://github.com/jetify-com/devbox.git go.jetify.com/devbox
       cd go.jetify.com/devbox
       go build ./cmd/devbox
       ./devbox run -- echo hello, world

### Debugging Devbox Locally

Several Devbox commands (for example `devbox shell` and `devbox services up`)
do little work themselves. Instead, they re-exec a nested `devbox` subprocess
inside the project's shell environment, and that subprocess does the real work.
The nested process is resolved from your `PATH`, so it is usually the *installed*
version of Devbox rather than the local build you are editing. As a result,
`devbox run build && dist/devbox services up` may not pick up your changes, and
debug logs (`DEVBOX_DEBUG=1`) or breakpoints you add won't fire, because the
interesting work happens in a different binary.

There are two ways to work around this:

1. **Make the launcher use your local build (most faithful).** The Devbox
   launcher selects which CLI binary to run based on the `DEVBOX_USE_VERSION`
   environment variable, looking it up in its binary cache. Place your build
   there and point the launcher at it:

       devbox run build
       cache_dir="${XDG_CACHE_HOME:-$HOME/.cache}/devbox/bin/0.0.0-dev_$(go env GOOS)_$(go env GOARCH)"
       mkdir -p "$cache_dir"
       cp dist/devbox "$cache_dir/devbox"
       export DEVBOX_USE_VERSION=0.0.0-dev

   With `DEVBOX_USE_VERSION` exported, every `devbox` invocation — including the
   nested subprocesses — runs your local build, so debug logs and breakpoints
   work end to end. Rebuild and re-copy the binary whenever you change the code.

2. **Run the work in the current process (quick).** Many commands accept a
   hidden `--run-in-current-shell` flag that skips the nested re-exec and does
   the work in place:

       dist/devbox services up --run-in-current-shell

   This is fast to iterate on, but because it bypasses the nested subprocess its
   behavior differs slightly from a normal invocation, so some issues won't
   reproduce this way.

## Pull Request Process

1. For new features or non-trivial changes, consider first filing an issue to
   discuss what changes you intend to make. This will let us help you with
   implementation details and to make sure we don't duplicate any work.
2. Ensure any new feature or functionality includes tests to verify its
   correctness.
3. Run `devbox run lint` and `devbox run test`.
4. Run `go mod tidy` if you added any new dependencies.
5. Submit your pull request and someone will take a look!

### Style Guide

We don't expect you to read through a long style guide or be an expert in Go
before contributing. When necessary, a reviewer will be happy to help out with
any suggestions around code style when you submit your PR. Otherwise, the Devbox
codebase generally follows common Go idioms and patterns:

- If you're unfamiliar with idiomatic Go,
  [Effective Go](https://go.dev/doc/effective_go) and the
  [Google Go Style Guide](https://google.github.io/styleguide/go) are good
  resources.
- There's no strict commit message format, but a good practice is to start the
  subject with the name of the Go packages you add/modified. For example,
  `boxcli: update help for add command`.

## Community Contribution License

Contributions made to this project must be made under the terms of the
[Apache 2 License](https://www.apache.org/licenses/LICENSE-2.0).

```
By making a contribution to this project, you certify that:

  a. The contribution was created in whole or in part by you and you have the right
  to submit it under the Apache 2 License; or

  b. The contribution is based upon previous work that, to the best of your
  knowledge, is covered under an appropriate open source license and you have the
  right under that license to submit that work with modifications, whether
  created in whole or in part by you, under the Apache 2 License; or

  c. The contribution was provided directly to you by some other person who
  certified (a), (b) or (c) and you have not modified it.

  d. You understand and agree that this project and the contribution are public
  and that a record of the contribution (including all personal information you
  submit with it, including your sign-off) is maintained indefinitely and may be
  redistributed consistent with this project or the open source license(s)
  involved.
```
