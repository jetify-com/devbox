# devbox create

Initialize a directory as a devbox project using a template

## Synopsis

Initialize a directory as a devbox project. This will create an empty devbox.json in the current directory. You can then add packages using `devbox add`

```bash
devbox create [dir] --template <template> [flags]
```

## List of templates

* [**go**](https://github.com/jetpack-io/devbox/tree/main/examples/development/go)
* [**node-npm**](https://github.com/jetpack-io/devbox/tree/main/examples/development/nodejs/nodejs-npm/)
* [**node-typescript**](https://github.com/jetpack-io/devbox/tree/main/examples/development/nodejs/nodejs-typescript/)
* [**node-yarn**](https://github.com/jetpack-io/devbox/tree/main/examples/development/nodejs/nodejs-yarn/)
* [**php**](https://github.com/jetpack-io/devbox/tree/main/examples/development/php/)
* [**python-pip**](https://github.com/jetpack-io/devbox/tree/main/examples/development/python/pip/)
* [**python-pipenv**](https://github.com/jetpack-io/devbox/tree/main/examples/development/python/pipenv/)
* [**python-poetry**](https://github.com/jetpack-io/devbox/tree/main/examples/development/python/poetry/)
* [**ruby**](https://github.com/jetpack-io/devbox/tree/main/examples/development/ruby/)
* [**rust**](https://github.com/jetpack-io/devbox/tree/main/examples/development/rust/)


## Options

<!--Markdown Table of Options  -->
| Option | Description |
| --- | --- |
| `-h, --help` | help for init |
| `-t, --template string` | Template to use for the project.|
| `-q, --quiet` | Quiet mode: Suppresses logs. |

## SEE ALSO

* [devbox](./devbox.md)	 - Instant, easy, predictable shells and containers

