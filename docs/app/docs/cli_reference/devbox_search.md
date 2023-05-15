# devbox search

Search for Nix packages

## Synopsis

`devbox search` will return a list of packages and versions that match your search query.

You can add a package to your project using `devbox add <package>`.

Too add a specific version, use `devbox add <package>@<version>`.

```bash
devbox search <pkg> [flags]
```

## Example

```bash
$ devbox search ripgrep

Warning: Search is experimental and may not work as expected.

Found 8+ results for "ripgrep":

* ripgrep (13.0.0, 12.1.1, 12.0.1)
* ripgrep-all (0.9.6, 0.9.5)

# To add ripgrep 12.1.1 to your project:

$ devbox add ripgrep@12.1.1
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-h, --help` | help for shell |
| `-q, --quiet` | Quiet mode: Suppresses logs. |

## SEE ALSO

* [devbox](./devbox.md)	 - Instant, easy, predictable shells and containers

