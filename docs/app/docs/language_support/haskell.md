---
title: Haskell
---

### Detection

Devbox will automatically create a Haskell project plan whenever a `stack.yaml` file is detected in the project's root directory.

For now, the Haskell planner only works for the [Stack framework](https://docs.haskellstack.org/en/stable/).

### Supported Versions

Devbox currently installs the latest Glasgow Haskell Compiler version 9.

### Included Nix Packages

* `ghc`
* `stack`
* `libiconv`
* `libffi`
* `binutils`

### Default Stages
These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details.
#### Install Stage

```bash
stack build --system-ghc --dependencies-only
```

#### Build Stage

```bash
stack build --system-ghc
```

#### Start Stage

```bash
stack exec --system-ghc <project name>-exe
```
where `<project name>` is from the `Name` field in `stack.yaml`
