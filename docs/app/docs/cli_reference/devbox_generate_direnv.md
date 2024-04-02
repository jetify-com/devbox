# devbox generate direnv

Top level command for generating the .envrc file for your Devbox Project. This can be used with [direnv](../ide_configuration/direnv.md) to automatically start your shell when you cd into your devbox directory

```bash
devbox generate direnv [flags]
```

## Options

<!-- Markdown table of options -->
| Option | Description |
| --- | --- |
| `-c, --config string` | path to directory containing a devbox.json config file |
|  `-e, --env stringToString` |  environment variables to set in the devbox environment (default []) |
|  `--env-file string` | path to a file containing environment variables to set in the devbox environment. If the file does not exist, then this parameter is ignored |
| `-h, --help` | help for generate |
| `-q, --quiet` | Quiet mode: Suppresses logs. |

## SEE ALSO

* [devbox generate](devbox_generate.md)	 - Generate supporting files for your project
