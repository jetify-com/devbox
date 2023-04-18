# devbox completion zsh

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

```bash
echo "autoload -U compinit; compinit" >> ~/.zshrc
```

To load completions in your current shell session:

```bash
source <(devbox completion zsh); compdef _devbox devbox
```

To load completions for every new session, execute once:

## Linux

```bash
devbox completion zsh > "${fpath[1]}/_devbox"
```

## macOS

```bash
devbox completion zsh > $(brew --prefix)/share/zsh/site-functions/_devbox
```

You will need to start a new shell for this setup to take effect.


```text
devbox completion zsh [flags]
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-h, --help` | help for zsh |
| `--no-descriptions` | disable completion descriptions |
| `-q, --quiet` | suppresses logs |


## SEE ALSO

* [devbox completion](devbox_completion.md)	 - Generate the autocompletion script for the specified shell

