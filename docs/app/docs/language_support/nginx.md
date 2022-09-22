---
title: Nginx
---

### Detection

Devbox will automatically create a Nginx Build plan whenever `nginx.conf`
 or `shell-nginx.conf` is detected in the project's root directory.

## Usage Notes

To run nginx in your shell, you can use the `shell-nginx` wrapper. This wrapper calls nginx with a few options. If you want to see what this wrapper does, use `cat $(which shell-nginx)`

While using Devbox shell, you should avoid pointing to assets or files outside the devbox.json directory because Nix might not have access. For example, you may want to keep your static files in `./static` within your project directory, instead of in another folder.

We generate a helper config `.devbox/gen/shell-helper-nginx.conf` that you can include in your `shell-nginx.conf` that sets a few defaults to ensure nginx can run in a nix shell. It should be included in the `server.http` block.

### Supported Versions

Devbox will use the nginx provided by the Nix Package manager. It is currently not possible to override the version.

### Included Nix Packages

- `nginx`

### Default Stages

These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details

### Install Stage

*No install stage for nginx planner*

### Build Stage

*No build stage for nginx planner*

### Start Stage

```bash
nginx -c <your_config> -g 'daemon off;'
```