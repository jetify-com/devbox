# Rust Example

A Devbox Shell for running Rust

## Configuration

This project adds the `rustup` package to your devbox shell, and then uses that package to install the right Rust toolchain locally in your project directory (set by the `RUSTUP_HOME` environment variable in `conf/set-env.sh`).

To change the version of Rust you want to use, you should modify the `rustup default stable` line in the `init_hook` of this project's `devbox.json`.

## How to Run

* Build the project

```bash
cargo build
```

* Run the project

```bash
cargo run
```
