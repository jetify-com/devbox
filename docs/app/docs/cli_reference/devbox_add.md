# devbox add

Add a new package to your devbox

```bash
devbox add <pkg>... [flags]
```

## Examples

```bash
# Add the latest version of the `ripgrep` package
devbox add ripgrep

# Install glibcLocales only on x86_64-linux and aarch64-linux
devbox add glibcLocales --platform x86_64-linux,aarch64-linux

# Exclude busybox from installation on macOS
devbox add busybox --exclude-platform aarch64-darwin,x86_64-darwin

# Install non-default outputs for a package, such as the promtool CLI
devbox add prometheus --outputs=out,cli
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `--allow-insecure` | allows Devbox to install a package that is marked insecure by Nix |
| `-c, --config string` | path to directory containing a devbox.json config file |
| `--disable-plugin` | disable the build plugin for a package |
| `--environment string` | Jetify Secrets environment to use, when supported (e.g.secrets support dev, prod, preview.) (default "dev") |
| `-e, --exclude-platform strings` | exclude packages from a specific platform. |
| `-h, --help` | help for add |
| `-o, --outputs strings` | specify the outputs to install for the nix package | 
| `-p`, `--platform strings` | install packages only on specific platforms. |
|  `--patch-glibc` | Patches ELF binaries to use a newer version of `glibc` |
| `-q, --quiet` | quiet mode: Suppresses logs. |

Valid Platforms include:

* `aarch64-darwin`
* `aarch64-linux`
* `x86_64-darwin`
* `x86_64-linux`

The platforms below are also supported, but will build packages from source

* `i686-linux`
* `armv7l-linux`


## SEE ALSO

* [devbox](./devbox.md)	 - Instant, easy, predictable shells and containers

