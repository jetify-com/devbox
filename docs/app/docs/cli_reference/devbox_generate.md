# devbox generate

Top level command for generating Devcontainer and Dockerfiles for your Devbox Project. 

```bash
devbox generate <devcontainer|dockerfile|direnv> [flags]
```

## Options

<!-- Markdown table of options -->
| Option | Description |
| --- | --- |
| `-c, --config string` | path to directory containing a devbox.json config file |
| `-h, --help` | help for generate |
| `-q, --quiet` | Quiet mode: Suppresses logs. |

## Subcommands

* [devbox generate devcontainer](devbox_generate_devcontainer.md)	 - Generate Dockerfile and devcontainer.json files under .devcontainer/ directory
* [devbox generate dockerfile](devbox_generate_dockerfile.md)	 - Generate a Dockerfile that replicates devbox shell
* [devbox generate direnv](devbox_generate_direnv.md)  - Generate a .envrc file to use with direnv

## SEE ALSO

* [devbox](devbox.md)	 - Instant, easy, predictable development environments

