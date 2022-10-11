# devbox shell

Start a new shell or run a command with access to your packages

## Synopsis

Start a new shell or run a command with access to your packages. 
If invoked without `cmd`, this will start an interactive shell based on the devbox.json in your current directory, or the directory provided with `dir`. 
If invoked with a `cmd`, this will start a shell based on the devbox.json provided in `dir`, run the command, and then exit.

```bash
devbox shell [<dir>] -- [<cmd>] [flags]
```

## Options

```text
  --print-env  Print a script to setup a devbox shell environment
  -h, --help   help for shell
```

## SEE ALSO

* [devbox](./devbox.md)	 - Instant, easy, predictable shells and containers

