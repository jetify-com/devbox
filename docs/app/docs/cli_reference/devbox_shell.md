## devbox shell

Start a new shell with access to your packages

### Synopsis

Start a new shell with access to your packages.

If the --config flag is set, the shell will be started using the devbox.json found in the --config flag directory. If --config isn't set, then devbox recursively searches the current directory and its parents.

```
devbox shell [flags]
```

### Options

```
  -c, --config string   path to directory containing a devbox.json config file
  -h, --help            help for shell
      --print-env       print script to setup shell environment
```

### Options inherited from parent commands

```
  -q, --quiet   suppresses logs
```

### SEE ALSO

* [devbox](devbox.md)	 - Instant, easy, predictable development environments

