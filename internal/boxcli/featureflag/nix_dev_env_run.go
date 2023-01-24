package featureflag

// NixDevEnvRun controls the implementation of `devbox run`. When enabled, `devbox run`
// runs the script in the environment returned by `nix print-dev-env`. This means the
// environment is much more "strict" or "pure", since it will _not_ include parts of
// the host's environment like `devbox shell` does.
var NixDevEnvRun = disabled("NIX_DEV_ENV_RUN")
