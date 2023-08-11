# devbox services restart

Restarts service. If no service is specified, restarts all services and process-compose.

```bash
devbox services restart [service]... [flags]
```

:::info
  Note: We recommend using `devbox services up` if you are starting all your services and process-compose. This command lets you specify your process-compose file and whether to run process-compose in the foreground or background.
:::

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
|  `-e, --env stringToString` |  environment variables to set in the devbox environment (default []) |
|  `--env-file string` | path to a file containing environment variables to set in the devbox environment |
| `-h, --help` | help for restart |
| `-q, --quiet` | Quiet mode: Suppresses logs. |

## SEE ALSO

* [devbox services](devbox_services.md)	 - Interact with devbox services

