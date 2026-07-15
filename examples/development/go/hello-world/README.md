# Go

Go projects can be run in Devbox by adding the Go SDK to your project. If your project uses cgo or compiles against C libraries, you should also include them in your packages to ensure Go can compile successfully

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/go/hello-world)


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

## Setting up the Go environment

This example configures a few environment variables in `devbox.json`:

```json
"env": {
    "GOPATH": "$HOME/go/",
    "PATH":   "$PATH:$HOME/go/bin"
}
```

- `GOPATH` is set to `$HOME/go`, which is Go's default location. Keeping the
  `GOPATH` outside of your project directory lets the module cache
  (`$GOPATH/pkg/mod`) be shared across projects and keeps build artifacts out of
  your source tree.
- Adding `$GOPATH/bin` to `PATH` makes tools installed with `go install`
  available inside the Devbox shell.

The `init_hook` exports `GOROOT` so that the Go toolchain provided by Nix is
used:

```json
"shell": {
    "init_hook": [
        "export \"GOROOT=$(go env GOROOT)\""
    ]
}
```

> **Note on `GOPATH`:** avoid setting `GOPATH` to your project directory (e.g.
> `"GOPATH": "$PWD"`). When a module (a directory containing `go.mod`) lives
> inside `GOPATH`, the Go toolchain prints `go: warning: ignoring go.mod in
> $GOPATH` and falls back to legacy `GOPATH` mode. Using `$HOME/go` (as above)
> keeps module-aware builds working as expected.
