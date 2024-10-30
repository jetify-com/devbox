# NodeJS

Most NodeJS Projects will install their dependencies locally using NPM or Yarn, and thus can work with Devbox with minimal additional configuration. Per project packages can be managed via NPM or Yarn.

[![Open In Devspace](https://www.jetify.com/img/devbox/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/node-npm)

## Adding NodeJS to your Shell

`devbox add nodejs`, or in your `devbox.json`:

```json
  "packages": [
    "nodejs@18"
  ],
```

This will install NodeJS 18, and comes bundled with `npm`. You can find other installable versions of NodeJS by running `devbox search nodejs`. You can also view the available versions on [Nixhub](https://www.nixhub.io/packages/nodejs)

## Adding Yarn as your Package Manager

`devbox add yarn`, or in your `devbox.json` add:

```json
  "packages": [
    "nodejs@18",
    "yarn@latest"
  ],
```

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
