# devbox completion bash

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

```bash
	source <(devbox completion bash)
```

To load completions for every new session, execute once:

## Linux

```bash
	devbox completion bash > /etc/bash_completion.d/devbox
```

## macOS

```bash
	devbox completion bash > $(brew --prefix)/etc/bash_completion.d/devbox
```

You will need to start a new shell for this setup to take effect.


```bash
devbox completion bash
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-h, --help` | help for bash |
| `--no-descriptions` | disable completion descriptions |
| `-q, --quiet` | suppresses logs |

## SEE ALSO

* [devbox completion](devbox_completion.md)	 - Generate the autocompletion script for the specified shell

