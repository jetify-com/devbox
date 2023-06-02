---
title: F# and .NET
---

F# and .NET projects can be easily generated in Devbox by adding the dotnet SDK to your project. You can then create new projects using `dotnet new`

[**Example Repo**](https://github.com/jetpack-io/devbox/tree/main/examples/development/fsharp)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/new?template=fsharp)

## Adding .NET to your project

`devbox add dotnet-sdk`, or add the following to your `devbox.json`:

```json
  "packages": [
    "dotnet-sdk@latest"
  ],
```
This will install the latest version of the dotnet SDK. You can find other installable versions of the dotnet SDK by running `devbox search dotnet-sdk`.

## Creating a new F# Project

`dotnet new console -lang "F#" -o <name>`
