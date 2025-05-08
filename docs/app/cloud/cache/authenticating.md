---
title: Authenticating with the Cache
sidebar_position: 2
---

Your Jetify Cloud organization is automatically provisioned with a shared cache. Any developers invited to your Jetify Cloud org will be automatically authenticated with the cache when they sign in. 

## Managing Access to the Cache

Team members can be added in one of two Roles, which controls their access to the Jetify Build Cache. 

- **Members** have read-only access to the cache, and cannot push new packages
- **Admins** have full read/write access to the cache, and can push new packages.

You can add or remove team members from your team, or modify their role, using the [Jetify Cloud Dashboard](../dashboard/inviting_members.md)

## Authenticating from the CLI

Once you’ve been invited to a team, you can authenticate from the CLI by running: 

```bash
devbox auth login
```

This will launch a browser window where you can authenticate with an email address or via Google SSO. 

You can check your current authentication status by running: 

```bash
devbox auth whoami
```

You can check that you are connected to the cache, and your current cache URL, by running: 

```bash
devbox cache info
```

You can logout by running:

```bash
devbox auth logout
```

### Authenticating CI or Build Hosts

Admin users can generate Personal Access Tokens to authenticate on hosts where you cannot login via the CLI or Browser. This token will have the same push/pull permissions as the account that generated it.

:::warning
Treat your Personal Access Token as a password — keep it secret and secure, and do not share it with other users.
:::

To generate a Token, first authenticate as described above, and then run:

```bash
devbox auth tokens new
```

To authenticate with the personal access token, export it as an environment variable on your host: 

```bash
export DEVBOX_API_TOKEN=<personal_token>
```
