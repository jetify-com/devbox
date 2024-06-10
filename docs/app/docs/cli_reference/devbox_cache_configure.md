# devbox cache configure

Configure Nix to use the Devbox cache as a substituter.

If the current Nix installation is multi-user, this command grants the Nix
daemon access to Devbox caches by making the following changes:

- Adds the current user to Nix's list of trusted users in the system nix.conf.
- Adds the cache credentials to ~root/.aws/config.

Configuration requires sudo, but only needs to happen once. The changes persist
across Devbox accounts and organizations.

This command is a no-op for single-user Nix installs that aren't running the
Nix daemon.

```bash
  devbox cache configure [flags]
```

## Options

<!-- Markdown table of options -->
| Option | Description |
| --- | --- |
| `--user string` | The OS user to configure Nix for. Defaults to the current user. |
| `-h, --help` | help for configure |
| `-q, --quiet` | suppresses logs |
