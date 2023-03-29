## devbox completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	devbox completion fish | source

To load completions for every new session, execute once:

	devbox completion fish > ~/.config/fish/completions/devbox.fish

You will need to start a new shell for this setup to take effect.


```
devbox completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -q, --quiet   suppresses logs
```

### SEE ALSO

* [devbox completion](devbox_completion.md)	 - Generate the autocompletion script for the specified shell

