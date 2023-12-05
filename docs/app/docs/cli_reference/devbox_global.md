# devbox global

Top level command for managing global packages.

You can use `devbox global` to install packages that you want to use across all your local devbox projects. For example -- if you usually use `ripgrep` for searching in all your projects, you can use `devbox global add ripgrep` to make it available whenever you start a `devbox shell` without adding it to each project's `devbox.json.` 

You can also use Devbox as a global package manager by adding the following line to your shellrc: 

`eval "$(devbox global shellenv)"`

For more details, see [Use Devbox as your Primary Package Manager](../devbox_global.md).

```bash
devbox global <subcommand> [flags]
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-c, --config string` | path to directory containing a devbox.json config file |
| `-h, --help` | help for generate |
| `-q, --quiet` | Quiet mode: Suppresses logs. |

## Subcommands
* [devbox global add](devbox_global_add.md)	 - Add a global package to your devbox
* [devbox global list](devbox_global_list.md)	 - List global packages
* [devbox global pull](devbox_global_pull.md)	 - Pulls a global config from a file or URL.
* [devbox global rm](devbox_global_rm.md)	 - Remove a global package 
* [devbox global shellenv](devbox_global_shellenv.md)	 - Print shell commands that add global Devbox packages to your PATH

## SEE ALSO

* [devbox](devbox.md)	 - Instant, easy, predictable development environments
