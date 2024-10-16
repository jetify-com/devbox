---
title: Zig
---

Zig projects can be run in Devbox by adding Zig and Nimble to your project.

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/zig/zig-hello-world)

[![Open In Devspace](../../../static/img/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/zig)

## Adding Zig to your Project

`devbox add zig`, or add the following to your `devbox.json`

```json
    "packages": [
        "zig@latest",
    ]
```

This will install the latest version of Zig. You can find other installable versions of Zig by running `devbox search zig`. You can also view the available versions on [Nixhub](https://www.nixhub.io/packages/zig)
