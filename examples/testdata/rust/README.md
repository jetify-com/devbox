To create a new Rust test case project, do: `cargo new <name>`.

The version checks in `rust-stable` and `rust-1.62.0` require us to add `rustc` to the packages in the runtime image produced from `docker build`.

The `rust-stable-hello-world` project has been added in order to sanity check the default runtime image.
