---
title: Haskell
---

Haskell projects that use the Stack Framework can be run in Devbox by adding the Stack and the Cabal packages to your project. You may also want to include libraries that Stack requires for compilation (described below)

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/development/haskell/)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/github.com/jetpack-io/devbox-examples?folder=development/development/haskell)

## Adding Haskell and Stack to your Project

`devbox add stack cabal-install zlib hpack`, or add the following to your `devbox.json`

```json
  "packages": [
    "stack",
    "cabal-install",
    "zlib",
    "hpack"
  ]
```

This will install GHC, and the Haskell Tool Stack in your Devbox Shell.