# devbox completion fish

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

```bash
devbox completion fish | source
```

To load completions for every new session, execute once:

```bash
devbox completion fish > ~/.config/fish/completions/devbox.fish
```

You will need to start a new shell for this setup to take effect.


```bash
devbox completion fish [flags]
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-h, --help` | help for fish |
| `--no-descriptions` | disable completion descriptions |
| `-q, --quiet` | suppresses logs |

## SEE ALSO

* [devbox completion](devbox_completion.md)	 - Generate the autocompletion script for the specified shell

