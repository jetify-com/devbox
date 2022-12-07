---
title: NodeJS
---

Most NodeJS Projects will install their dependencies locally using NPM or Yarn, and thus can work with Devbox with minimal additional configuration. Per project packages can be managed via NPM or Yarn.

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/development/nodejs)

## Adding NodeJS to your Shell

`devbox add nodejs`, or in your `devbox.json`:
```json
  "packages": [
    "nodejs"
  ],
```

This will install NodeJS 18, and comes bundled with `npm`. 

Other versions available include: 

* `nodejs-16_x` (NodeJS 16)
* `nodejs-19_x` (NodeJS 19)

## Adding Yarn as your Package Manager

`devbox add yarn`, or in your `devbox.json` add: 
```json
  "packages": [
    "nodejs",
    "yarn"
  ],
```

## Installing Global Packages

In some situations, you may want to install packages using `npm install --global`. This will fail in Devbox since the Nix Store is immutable. 

You can instead install these global packages by adding them to the list of packages in your `devbox.json`. For example: to add `yalc` and `pm2`: 

```json
{
    "packages": [
        "nodejs",
        "nodePackages.yalc",
        "nodePackages.pm2"
    ]
}
```