# Bun

Bun projects can be run in Devbox by adding the Bun runtime + package manager to your project.

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/bun)

[![Open In Devspace](https://www.jetify.com/img/devbox/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/bun)

## Add Bun to your Project

```bash
devbox add bun@latest
```

You can see which versions of `bun` are available using:

```bash
devbox search bun
```

To update bun to the latest version:

```bash
devbox update bun
```

## Scripts

To install dependencies:

```bash
devbox run bun install
```

To start + watch your project:

```bash
devbox run dev
```

This project was created using `bun init` in bun v1.0.33. [Bun](https://bun.sh) is a fast all-in-one JavaScript runtime.
