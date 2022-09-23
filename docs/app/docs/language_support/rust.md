---
title: Rust
---
### Detection

Devbox will automatically create a Rust Build plan whenever `Cargo.toml` is detected in the project's root directory.

### Supported Versions

Devbox uses the [nix-overlay from oxalica](https://github.com/oxalica/rust-overlay). The oldest version it supports is `1.29.0` and the latest is `1.63.0` (as of September 19, 2022)

### Included Nix Packages

- Devbox uses a [nix-overlay from oxalica](https://github.com/oxalica/rust-overlay) in the Shell and Development image.
    - Using this overlay, Devbox can install nix packages for the version of the rust toolchain specified in `rust-version` in `Cargo.toml` . The nix package is `rust-bin.stable.<rust-version>.default`
    - If no `rust-version` is specified, it uses the latest version of Rust.  Nix package is `rust-bin.stable.latest.default`
- All other Packages Installed:
    - Shell and Development Image: `gcc`
    - Runtime Image: `glibc`

### Default Stages

These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details

### Install Stage

```bash
cargo fetch
```

### Build Stage

```bash
cargo build --release --offline
```

### Start Stage

```bash
# <package-name> is the executable name 
# derived from package.name field in Cargo.toml
./<package-name>
```