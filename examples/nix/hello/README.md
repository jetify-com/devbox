# Custom Nix Package Example

This example shows how to include a locally-authored Nix expression in your
devbox shell. The `hello-pkg/default.nix` file defines a trivial shell script
using `pkgs.writeShellScriptBin`, and devbox consumes it via its existing
local-flake pipeline.

## One-time scaffolding

devbox's package syntax expects a flake, so the first time you set this up
(or whenever you change the pinned nixpkgs), generate a thin wrapper flake
next to the `default.nix`:

```sh
devbox generate flake-wrapper ./hello-pkg
```

This writes `hello-pkg/flake.nix` (see below). The wrapper is intentionally
not committed to this example repo so you can see the scaffolding step.

## devbox.json

```json
{
  "packages": {
    "./hello-pkg": ""
  }
}
```

The `./hello-pkg` entry is a standard local-flake reference: devbox passes it
to Nix as `path:./hello-pkg` and adds `packages.${system}.default` from that
flake to the shell's `buildInputs`.

## Running the example

```sh
devbox generate flake-wrapper ./hello-pkg
devbox shell -- hello
# Hello from a custom Nix package!
```

## What the generated wrapper looks like

`devbox generate flake-wrapper ./hello-pkg` produces something like:

```nix
{
  description = "devbox wrapper flake for hello-pkg";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }: let
    forAllSystems = f: nixpkgs.lib.genAttrs
      nixpkgs.lib.systems.flakeExposed
      (system: f nixpkgs.legacyPackages.${system});
  in {
    packages = forAllSystems (pkgs: {
      default = pkgs.callPackage ./default.nix {};
    });
  };
}
```

`pkgs.callPackage ./default.nix {}` auto-injects the usual nixpkgs arguments
(`stdenv`, `lib`, `bash`, ...) into the expression. You can hand-edit the
wrapper to pass overrides (e.g. `pkgs.callPackage ./default.nix { withSsl = true; }`),
expose additional attributes, or pin a different nixpkgs — and re-run
`devbox generate flake-wrapper --force ./hello-pkg` whenever you want to
regenerate it from scratch.
