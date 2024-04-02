# devbox generate dockerfile

Generate a Dockerfile that replicates devbox shell

## Synopsis

Generate a Dockerfile that replicates devbox shell. Can be used to run devbox shell environment in an OCI container.

```bash
devbox generate dockerfile [flags]
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-c, --config string` | path to directory containing a devbox.json config file |
| `-f, --force` | force overwrite existing files |
| `--root-user` | use `root` as the user for container. Installs nix as single-user mode in Dockerfile |
| `-h, --help` | help for dockerfile |
| `-q, --quiet` | Quiet mode: Suppresses logs. |


## SEE ALSO

* [devbox generate](devbox_generate.md)	 - Generate supporting files for your project
