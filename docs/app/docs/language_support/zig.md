---
title: Zig
---

### Detection

Devbox will automatically create a Zig project plan whenever a `build.zig` file is detected in the project's root directory.

### Supported Versions

Devbox currently installs the latest Zig version.

### Included Nix Packages

Install and Build Stage Image:
* `zig`

Start Stage Image:
* None, if we can parse the executable name from `build.zig`.
* `zig`, otherwise.

### Default Stages
These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details.
#### Install Stage

Skipped: *No install stage for zig planner*

#### Build Stage

```bash
zig build install
```

#### Start Stage

If `<exe-name>` is found from parsing `build.zig` to find the `addExecutable` statement, then:
```bash
./<exe-name>
```

Else if no `<exe-name>` is found, then:
```bash
zig build run
```
