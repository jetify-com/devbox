---
title: F# and .NET
---

F# and .NET projects can be easily generated in Devbox by adding the dotnet SDK to your project. You can then create new projects using `dotnet new`

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/fsharp)

[![Open In Devspace](../../../static/img/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/fsharp)

## Adding .NET to your project

`devbox add dotnet-sdk`, or add the following to your `devbox.json`:

```json
  "packages": [
    "dotnet-sdk@latest"
  ],
```

This will install the latest version of the dotnet SDK. You can find other installable versions of the dotnet SDK by running `devbox search dotnet-sdk`. You can also view the available versions on [Nixhub](https://www.nixhub.io/search?q=dotnet)

## Creating a new F# Project

`dotnet new console -lang "F#" -o <name>`
