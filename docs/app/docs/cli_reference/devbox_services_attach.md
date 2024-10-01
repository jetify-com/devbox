# devbox services attach

Attach to a running instance of `devbox services`. This command lets you launch the TUI for process-compose if you started your services in the background with `devbox services up -b`.

Note that terminating the TUI will not stop your backgrounded services. To stop your services, use `devbox services stop`.

```bash
devbox services attach [flags]
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-h, --help` | help for ls |
| `-q, --quiet` | Quiet mode: Suppresses logs. |

### SEE ALSO

* [devbox services](devbox_services.md)	 - Interact with devbox services
