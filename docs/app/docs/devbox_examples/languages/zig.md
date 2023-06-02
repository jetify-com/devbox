---
title: Zig
---

Zig projects can be run in Devbox by adding Zig and Nimble to your project.

[**Example Repo**](https://github.com/jetpack-io/devbox/tree/main/examples/development/zig/zig-hello-world)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/new?template=zig)

## Adding Zig to your Project

`devbox add zig`, or add the following to your `devbox.json`

```json
    "packages": [
        "zig@latest",
    ]
```

This will install the latest version of Zig. You can find other installable versions of Zig by running `devbox search zig`.
