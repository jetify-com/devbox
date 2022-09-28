---
title: Go
---

### Detection

Devbox will automatically create a Go project plan whenever a `go.mod` file is detected in the project's root directory

### Supported Versions

Devbox will try to detect your Go version by looking for a `go` directive in your `go.mod`. Supported major versions include:

* 1.19
* 1.18
* 1.17

If no Go version can be detected, Devbox will default to 1.19.

### Included Nix Packages

* Depending on which version of Go is detected:
  * `go_1_17`
  * `go`
  * `go_1_19`

### Default Stages
These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details.
#### Install Stage 

```bash
go get
```

#### Build Stage 

```bash
CGO_ENABLED=0 go build -o app
```

#### Start Stage 

```bash
./app
```
