# devbox global run

Starts a new shell and runs your script or command in it, exiting when done.

The script must be defined in `devbox.json`, or else it will be interpreted as an arbitrary command. You can pass arguments to your script or command. Everything after `--` will be passed verbatim into your command (see example)

```bash
devbox global run <pkg>... [flags]
```

## Examples

Run a command directly:

```bash
  devbox add cowsay
  devbox global run cowsay hello
  devbox global run -- cowsay -d hello
```

Run a script (defined as `"moo": "cowsay moo"`) in your devbox.json:

```bash
  devbox global run moo
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
|  `-e, --env stringToString` |  environment variables to set in the devbox environment (default []) |
|  `--env-file string` | path to a file containing environment variables to set in the devbox environment |
| `-h, --help` | help for global run |
| `-q, --quiet` | suppresses logs |

## SEE ALSO

* [devbox global](devbox_global.md)	 - Manages global Devbox packages
