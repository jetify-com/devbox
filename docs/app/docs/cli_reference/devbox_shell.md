# devbox shell

Start a new shell or run a command with access to your packages

## Synopsis

Start a new shell or run a command with access to your packages. The interactive shell will use the devbox.json in your current directory, or the directory provided with `dir`.

```bash
devbox shell [<dir>] [flags]
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
|  `-e, --env stringToString` |  environment variables to set in the devbox environment (default []) |
|  `--env-file string` | path to a file containing environment variables to set in the devbox environment |
| `--print-env` | Print a script to setup a devbox shell environment |
| `--pure` | If this flag is specified, devbox creates an isolated shell inheriting almost no variables from the current environment. A few variables, in particular HOME, USER and DISPLAY, are retained. |
| `-h, --help` | help for shell |
| `-q, --quiet` | Quiet mode: Suppresses logs. |

## SEE ALSO

* [devbox](./devbox.md)	 - Instant, easy, predictable shells and containers

