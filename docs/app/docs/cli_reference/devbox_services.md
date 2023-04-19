# devbox services

Interact with Devbox services via process-compose

```bash
devbox services <ls|restart|start|stop> [flags]
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-c, --config string` | path to directory containing a devbox.json config file |
| `-h, --help` | help for services |
| `-q, --quiet` | Quiet mode: Suppresses logs. |

## Subcommands

* [devbox services ls](devbox_services_ls.md)	 - List available services
* [devbox services restart](devbox_services_restart.md)	 - Restarts service. If no service is specified, restarts all services
* [devbox services start](devbox_services_start.md)	 - Starts service. If no service is specified, starts all services
* [devbox services stop](devbox_services_stop.md)	 - Stops service. If no service is specified, stops all services

## SEE ALSO

* [devbox](devbox.md)	 - Instant, easy, predictable development environments
