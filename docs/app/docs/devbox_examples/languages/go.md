---
title: Go
---

Go projects can be run in Devbox by adding the Go SDK to your project. If your project uses cgo or compiles against C libraries, you should also include them in your packages to ensure Go can compile successfully

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/go/hello-world)

[![Open In Devspace](../../../static/img/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/go)

## Adding Go to your Project

`devbox add go`, or add the following to your `devbox.json`

```json
  "packages": [
    "go@latest"
  ]
```

This will install the latest version of the Go SDK. You can find other installable versions of Go by running `devbox search go`. You can also view the available versions on [Nixhub](https://www.nixhub.io/packages/go)

If you need additional C libraries, you can add them along with `gcc` to your package list. For example, if libcap is required for your project:

```json
"packages": [
    "go",
    "gcc",
    "libcap"
]
```

## Installing go packages that have CLIs

Installing go packages in your devbox shell is as simple as `go get <package_name>` but some packages come with a CLI of their own (e.g., `godotenv`). That means after installing the package you should be able to use the CLI binary and also control where that binary is installed. This is done by setting `$GOPATH` or `$GOBIN` env variable.

With Devbox you can set these variables in the `"env"` section of your `devbox.json` file.
In the example below we are setting `$GOPATH` to be the same directory of our project and therefore `godotenv` binary will be located in the `bin/` subdirectory of `$GOPATH`:

```json
{
  "packages": [
    "go@latest"
  ],
  "env": {
    "GOPATH": "$PWD",
    "PATH": "$PATH:$PWD/bin"
  },
  "shell": {
    "init_hook": [
      "echo 'Welcome to devbox!' > /dev/null"
    ],
    "scripts": {}
  }
}
```

Running `go install github.com/joho/godotenv/cmd/godotenv@latest` will create a `bin/` subdirectory in my project and puts `godotenv` there. Since I also added that subdirectory to my `$PATH`, my devbox shell can now recognize the `godotenv` binary and I can run commands like `godotenv -h` to use `godotenv` in CLI mode.
