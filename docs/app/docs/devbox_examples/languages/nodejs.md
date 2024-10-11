---
title: NodeJS
---

Most NodeJS Projects will install their dependencies locally using NPM or Yarn, and thus can work with Devbox with minimal additional configuration. Per project packages can be managed via NPM or Yarn.

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/nodejs)

[![Open In Devspace](../../../static/img/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/nodejs)

## Adding NodeJS to your Shell

`devbox add nodejs`, or in your `devbox.json`:

```json
  "packages": [
    "nodejs@18"
  ],
```

This will install NodeJS 18, and comes bundled with `npm`. You can find other installable versions of NodeJS by running `devbox search nodejs`. You can also view the available versions on [Nixhub](https://www.nixhub.io/packages/nodejs)

## Adding Yarn, NPM, or pnpm as your Node Package Manager

We recommend using [Corepack](https://github.com/nodejs/corepack/) to install and manage your Node Package Manager in Devbox. Corepack comes bundled with all recent Nodejs versions, and you can tell Devbox to automatically configure Corepack using a built-in plugin. When enabled, corepack binaries will be installed in your project's `.devbox` directory, and automatically added to your path.

To enable Corepack, set `DEVBOX_COREPACK_ENABLED` to `true` in your `devbox.json`:

```json
{
  "packages": ["nodejs@18"],
  "env": {
    "DEVBOX_COREPACK_ENABLED": "true"
  }
}
```

To disable Corepack, remove the `DEVBOX_COREPACK_ENABLED` variable from your devbox.json

### Yarn

[**Example Repo**](https://github.com/jetify-com/devbox?folder=examples/development/nodejs/nodejs-yarn)

[![Open In Devspace](../../../static/img/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/nodejs-yarn)

### pnpm

[**Example Repo**](https://github.com/jetify-com/devbox?folder=examples/development/nodejs/nodejs-pnpm)

[![Open In Devspace](../../../static/img/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/nodejs-pnpm)

## Installing Global Packages

In some situations, you may want to install packages using `npm install --global`. This will fail in Devbox since the Nix Store is immutable.

You can instead install these global packages by adding them to the list of packages in your `devbox.json`. For example: to add `yalc` and `pm2`:

```json
{
  "packages": [
    "nodejs@18",
    "nodePackages.yalc@latest",
    "nodePackages.pm2@latest"
  ]
}
```
