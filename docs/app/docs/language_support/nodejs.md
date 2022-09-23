---
title: NodeJS
---

### Detection

Devbox will automatically create a NodeJs Build plan whenever a `package.json` file is detected in the project's root directory.

### Supported Versions

Devbox will attempt to detect the Node version set in your `package.json` file. The following major versions are supported (Devbox will always use the latest minor version for each major version):

-   10
-   12
-   16
-   18

If no version is set, Devbox will use 16 as the default version

### Included Nix Packages

-   Depending on the detected Node Version:
    -   `nodejs`
    -   `nodejs10_x`
    -   `nodejs12_x`
    -   `nodejs16_x`
    -   `nodejs18_x`
-   If a `package-lock.json` is detected, or if no lockfile is detected:
    -   `npm`
-   If a `yarn.lock` file is detected:
    -   `yarn`

### Default Stages

These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details

### Install Stage

If a `package-lock.json` is detected:

```bash
npm install
```

If a `yarn.lock` is detected:

```bash
yarn install
```

Otherwise:

```bash
npm install
```

### Build Stage

This stage will copy over the rest of your project files.

If a `package-lock.json` is detected and a `build` script is found in `package.json`:

```bash
npm run build && npm prune --production
```

If a `yarn.lock` is detected and a `build` script is found in `package.json`:

```bash
yarn build && yarn install --production --ignore-scripts --prefer-offline
```

Otherwise, we simply remove the dev packages:

```bash
npm prune --production
```

### Start Stage

If a `package-lock.json` is detected and a `start` script is found in `package.json`:

```bash
npm start
```

If a `yarn.lock` is detected and a `start` script is found in `package.json`:

```bash
yarn start
```

Otherwise:

```bash
node index.js
```
