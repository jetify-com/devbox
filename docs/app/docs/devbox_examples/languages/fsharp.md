---
title: F# and .NET
---

F# and .NET projects can be easily generated in Devbox by adding the dotnet SDK to your project. You can then create new projects using `dotnet new`

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/development/fsharp)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/github.com/jetpack-io/devbox-examples?folder=development/fsharp/hello-world)

## Adding .NET to your project

`devbox add dotnet-sdk`, or add the following to your `devbox.json`:

```json
  "packages": [
    "dotnet-sdk"
  ],
```
This will install .NET SDK 6.0

Other versions available include: 

* dotnet-sdk_7 (version 7.0)
* dotnet-sdk_5 (version 5.0)
* dotnet-sdk_3 (version 3.1)

## Creating a new F# Project

`dotnet new console -lang "F#" -o <name>`