# Devbox Quickstart

This shell includes a basic `devbox.json` with a few useful packages installed, and an example init_hook and script

[![Open In Devbox.sh](https://www.jetify.com/img/devbox/open-in-devbox.svg)](https://devbox.sh/github.com/jetify-com/devbox-examples?folder=tutorial)

## Adding New Packages

Run `devbox add <package>` to add a new package. Remove it with `devbox rm <package>`.

For example: install Python 3.10 by running:

```bash
devbox add python310
```

Devbox can install over 80,000 packages via the Nix Package Manager. Search for packages at [https://search.nixos.org/packages](https://search.nixos.org/packages)

## Running Devbox Scripts

You can add new scripts by editing the `devbox.json` file

You can run scripts using `devbox run <script>`

For example: you can replay this help text with:

```bash
devbox run readme
```

## Next Steps

-   Checkout our Docs at [https://www.jetify.com/devbox/docs](https://www.jetify.com/devbox/docs)
-   Try out an Example Project at [https://www.jetify.com/devbox/docs/devbox-examples](https://www.jetify.com/devbox/docs/devbox-examples)
-   Report Issues at [https://github.com/jetify-com/devbox/issues/new/choose](https://github.com/jetify-com/devbox/issues/new/choose)
