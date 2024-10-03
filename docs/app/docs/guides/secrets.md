---
title: Managing Secrets and Environment Variables
---

Devbox provides a few different methods for including environment variables and secrets in your Devbox shell. This guide will walk you through the different methods and help you choose the best one for your use case.

## Setting Environment Variables in your Devbox Config

Environment variables that do not need to be stored securely can be set directly in your `devbox.json` under the `env` object. This is useful for non-secret variables that you want to have set in your Devbox shells:

```json
{
  "env": {
    "MY_VAR": "my_value"
  },
  "packages": {},
  "shell": {}
}
```

Currently, you can only set values using string literals, `$PWD`, and `$PATH`. Any other values with environment variables will not be expanded when starting your shell. For more details, see (/devbox/docs/configuration.md).

## Setting Environment Variables with Env Files

For environment variables that you want to keep out of your `devbox.json` file, you can use an env file. Env files are text files that contain key-value pairs, one per line. You can reference an env file in your `devbox.json` like this:

```json
{
  "packages": {},
  "shell": {},
  "env_from": [
    "path/to/.env"
  ]
}
```

## Securely Managing Secrets with Jetify Secrets

For secrets that need to be stored securely, you can use Jetify Secrets. Jetify Secrets is a secure secrets management service that allows you to store and manage your secrets with Jetify. You can then access your secrets whenever you start your Devbox shell, and manage them from the CLI using [`devbox secrets`](/devbox/docs/cli_reference/devbox_secrets).

To get started with Jetify Secrets, you will need to first create an account on Jetify Cloud and login with `devbox auth login`. Once your account is created, you can create a new project and start adding secrets to your project using `devbox secrets init`.

For more details on how to manage your secrets from the CLI, see our guide on [**Jetify Secrets**](/cloud/docs/secrets/secrets_cli).
