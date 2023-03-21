---
title: Go
---

Go projects can be run in Devbox by adding the Go SDK to your project. If your project uses cgo or compiles against C libraries, you should also include them in your packages to ensure Go can compile successfully

[**Example Repo**](https://github.com/jetpack-io/devbox/tree/main/examples/development/go/hello-world)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/github.com/jetpack-io/devbox?folder=examples/development/go/hello-world)

## Adding Go to your Project

`devbox add go`, or add the following to your `devbox.json`

```json
  "packages": [
    "go"
  ]
```

This will install go 1.18. 

Other versions available to install include: 

  * `go_1_19` (version 1.19)
  * `go_1_17` (version 1.17)
  

If you need additional C libraries, you can add them along with `gcc` to your package list. For example, if libcap is required for yoru project: 

```json
"packages": [
    "go",
    "gcc", 
    "libcap"
]
