# devbox run

Starts a new interactive shell and runs your target script in it. The shell will exit once your target script is completed or when it is terminated via CTRL-C. Scripts can be defined in your `devbox.json`.

You can also run arbitrary commands in your devbox shell by passing them as arguments to `devbox run`. For example:

```bash
  devbox run echo "Hello World"
```
Will print `Hello World` to the console from within your devbox shell.

For more details, read our [scripts guide](../guides/scripts.md)

```bash
  devbox run <script | command> [flags]
```


## Examples

```bash
# Run a command directly:
  devbox add cowsay
  devbox run cowsay hello
# Run a command that takes flags:
  devbox run cowsay -d hello
# Pass flags to devbox while running a command.
# All `devbox run` flags must be passed before the command
  devbox run -q cowsay -d hello

#Run a script (defined as `"moo": "cowsay moo"`) in your devbox.json:
  devbox run moo
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-c, --config string` | path to directory containing a devbox.json config file |
| `-e, --env stringToString` |  environment variables to set in the devbox environment (default []) |
| `--env-file string` | path to a file containing environment variables to set in the devbox environment |
| `-h, --help` | help for run |
| `-q, --quiet` | Quiet mode: Suppresses logs. |



## SEE ALSO

* [devbox](./devbox.md)	 - Instant, easy, predictable shells and containers

