---
title: Haskell
---

Haskell projects that use the Stack Framework can be run in Devbox by adding the Stack and the Cabal packages to your project. You may also want to include libraries that Stack requires for compilation (described below)

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/haskell/)

## Adding Haskell and Stack to your Project

`devbox add stack cabal-install zlib hpack`, or add the following to your `devbox.json`

```json
  "packages": [
    "stack@latest",
    "cabal-install@latest",
    "zlib@latest",
    "hpack@latest"
  ]
```

This will install GHC, and the Haskell Tool Stack in your Devbox Shell at their latest version. You can find other installable versions of Stack by running `devbox search <pkg>`.