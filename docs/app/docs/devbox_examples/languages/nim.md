---
title: Nim
---

Nim projects can be run in Devbox by adding Nim and Nimble to your project. For some platforms, Nimble may return an error if OpenSSL is not available, so we recommend including `openssl_1_1` in your packages as well

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/nim/spinnytest)

## Adding Nim to your Project

`devbox add nim nimble-unwrapped openssl_1_1`, or add the following to your `devbox.json`

```json
    "packages": [
        "nim@latest",
        "nimble-unwrapped@latest",
        "openssl_1_1@latest"
    ]
```

This will install the latest version of Nim available. You can find other installable versions of Nim by running `devbox search nim`.
