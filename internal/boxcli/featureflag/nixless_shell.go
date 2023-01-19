package featureflag

// NixlessShell controls the nixless shell feature. When enabled, `devbox shell`
// creates a shell without relying on `nix-shell`, but by setting the required environment
// variables and spawning a shell directly.
var NixlessShell = disabled("NIXLESS_SHELL")
