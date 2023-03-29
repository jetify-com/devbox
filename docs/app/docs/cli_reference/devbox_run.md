## devbox run

Run a script or command in a shell with access to your packages

### Synopsis

Start a new shell and runs your script or command in it, exiting when done.

The script must be defined in `devbox.json`, or else it will be interpreted as an arbitrary command. You can pass arguments to your script or command. Everything after `--` will be passed verbatim into your command (see examples).



```
devbox run [<script> | <cmd>] [flags]
```

### Examples

```

Run a command directly:

  devbox add cowsay
  devbox run cowsay hello
  devbox run -- cowsay -d hello

Run a script (defined as `"moo": "cowsay moo"`) in your devbox.json:

  devbox run moo
```

### Options

```
  -c, --config string   path to directory containing a devbox.json config file
  -h, --help            help for run
```

### Options inherited from parent commands

```
  -q, --quiet   suppresses logs
```

### SEE ALSO

* [devbox](devbox.md)	 - Instant, easy, predictable development environments

