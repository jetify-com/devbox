# Maelstrom

[![Open In Devbox.sh](https://jetify.com/img/devbox/open-in-devbox.svg)](https://devbox.sh/github.com/jetify-com/devbox-examples?folder=cloud_development/maelstrom)

A Devbox for running [Maelstrom](https://github.com/jepsen-io/maelstrom) Tests. Maelstrom is a testing library for toy distributed systems built by @aphyr, useful for learning the basics and principals of building distributed systems

You should also check out the [Fly.io Distributed Systems Challenge](https://fly.io/dist-sys/)

## Prerequisites

If you don't already have [Devbox](https://www.jetify.com/devbox/docs/installing_devbox/), you can install it by running the following command:

```bash
curl -s https://get.jetify.com/install.sh | bash
```

You can skip this step if you're running on Devbox.sh

## Usage

1. Install Maelstrom by running `devbox run install`. This should install Maelstrom 0.2.2 in a `maelstrom` subdirectory

1. cd into the `maelstrom` directory and run `./maelstrom` to verify everything is working

1. You can now follow the docs and run the tests in the Maelstrom Docs + Readme. You can use `glow` from the command line to browse the docs.

This shell includes Ruby 3.10 for running the Ruby Demos. To run demos in other languages, install the appropriate runtimes using `devbox add`. For example, to run the Python demos, use `devbox add python310`.
