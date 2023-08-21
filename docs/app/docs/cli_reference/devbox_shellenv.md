# devbox shellenv

Print shell commands that add Devbox packages to your PATH

```bash
devbox shellenv [flags]
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-c, --config string` | path to directory containing a devbox.json config file |
|  `-e, --env stringToString` |  environment variables to set in the devbox environment (default []) |
|  `--env-file string` | path to a file containing environment variables to set in the devbox environment |
| `--pure` | If this flag is specified, devbox creates an isolated environment inheriting almost no variables from the current environment. A few variables, in particular HOME, USER and DISPLAY, are retained. |
| `-h, --help` | help for shellenv |
| `-q, --quiet` | suppresses logs |


### SEE ALSO

* [devbox](devbox.md)	 - Instant, easy, predictable development environments
