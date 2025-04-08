# Website

This website is built using [Docusaurus](https://docusaurus.io/), a modern static website generator.


## Installation

```bash
cd docs/app     # from the devbox root directory
devbox shell    # optional, develop inside a devbox
yarn install    # run in devbox shell
```

### Local Development

```bash
yarn start
```

This command starts a local development server and opens up a browser window. Most changes are reflected live without having to restart the server.

### Build

```bash
yarn build
```

This command generates static content into the `build` directory and can be served using any static contents hosting service.

### Deployment

When a pull request is opened, it will automatically deploy via CICD to a preview.
When a pull request is merged, it will automatically deploy to production.
Check https://www.jetify.com/devbox/ after merge to see the latest changes.

# Devbox Documentation

Welcome to the Devbox documentation! This documentation is built using Docusaurus and contains all the information you need to get started with Devbox.

## Contributing to Documentation

We welcome contributions to our documentation! If you find any issues or would like to suggest improvements, please feel free to open a pull request.
