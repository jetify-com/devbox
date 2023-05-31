# devbox global update

Updates packages in your Devbox global config to the latest available version.

## Synopsis

If you provide this command with a list of packages, it will update those packages to the latest available version based on the version tag provided.

For example: if your global config has `python@3.11` in your package list, running `devbox update` will update to the latest patch version of `python 3.11`.

If no packages are provided, this command will update all the versioned packages to the latest acceptable version.

```bash
devbox update [pkg]... [flags]
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-h, --help` | help for update |
| `-q, --quiet` | suppresses logs |

## SEE ALSO

* [devbox global](devbox_global.md)	 - Manages global Devbox packages
