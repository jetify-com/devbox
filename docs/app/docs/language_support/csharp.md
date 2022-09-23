---
title: C# and .NET
---

### Detection

Devbox will automatically create a .NET Build plan whenever `.csproj` or `.fsproj` is detected in the project's root directory.


### Supported Versions

Devbox will attempt to detect the version set in `PropertyGroup.TargetFramework` field of the `.csproj` or `.fsproj` file. The following major versions are supported:

- .Net 7 (preview of next release)
- .Net 6 (current official release)
- .Net 5
- .Net 3

If no version is set, Devbox will use .NET 6 as the default version. Devbox will always use the latest minor version for each major version

### Included Nix Packages

- Depending on the detected SDK Version:
    - `dotnet-sdk_7`
    - `dotnet-sdk` (default, .NET 6)
    - `dotnet-sdk_5`
    - `netcoreapp3`
- All other Packages Installed:
    - none

### Default Stages

These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details

### Install Stage

```bash
dotnet restore --packages nuget-packages
```

### Build Stage

```bash
dotnet publish -c Publish --no-restore
```

### Start Stage

```bash
dotnet run -c Publish --no-build
```