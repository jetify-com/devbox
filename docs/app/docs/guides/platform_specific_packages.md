---
title: Installing Platform Specific Packages
---

At times, you may need to install a package or library that is only available on a specific platform. For example, you may want to install a package that is only available on Linux, while still using the same Devbox configuration on your Mac.

Devbox allows you to specify which platforms a package should be installed on using the `--platform` and `--exclude-platform` flags. When a package is added using these flags, it will be added to your `devbox.json`, but will only be installed when you run Devbox on a matching platform.

:::info

Specifying platforms for packages will alter your `devbox.json` in a way that is only compatible with **Devbox 0.5.12** and newer.

If you encounter errors trying to run a Devbox project with platform-specific packages, you may need to run `devbox version update`
:::

## Installing Platform Specific Packages

To avoid build or installation errors, you can tell Devbox to only install a package on specific platforms using the `--platform` flag when you run `devbox add`.

For example, to install the `busybox` package only on Linux platforms, you can run:

```bash
devbox add busybox --platform x86_64-linux,aarch64-linux
```

This will add busybox to your `devbox.json`, but will only install it when use devbox on a Linux machine. The packages section in your config will look like the following

```json
{
    "packages": {
        "busybox": {
            "version": "latest",
            "platforms": ["x86_64-linux", "aarch64-linux"]
        }
    }
}
```

## Excluding a Package from Specific Platforms

You can also tell Devbox to exclude a package from a specific platform using the `--exclude-platform` flag. For example, to avoid installing `ripgrep` on an ARM-based Mac, you can run:


```bash
devbox add ripgrep --exclude-platform aarch64-darwin
```

This will add ripgrep to your `devbox.json`, but will not install it when use devbox on an ARM-based Mac. The packages section in your config will look like the following:

```json
{
    "packages": {
        "ripgrep": {
            "version": "latest",
            "excluded_platforms": ["aarch64-darwin"]
        }
    }
}
```

## Supported Platforms

Valid Platforms include:

* `aarch64-darwin`
* `aarch64-linux`
* `x86_64-darwin`
* `x86_64-linux`

The platforms below are also supported, but will build packages from source

* `i686-linux`
* `armv7l-linux`