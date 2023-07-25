{
  description =
    "This flake outputs a modified version of Yarn that uses NodeJS 16";

  inputs = {
    nixpkgs.url = "nixpkgs/fc3de6da83863f8f36fdcac1c199c6066a6a0378";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
  # Use the flake-utils lib to easily create a multi-system flake
  flake-utils.lib.eachDefaultSystem (system:
    let
      # You can define overlays as functions using the example below
      # This overlay will modify yarn to use nodejs-16_x
      overlay = (final: prev: {
        yarn = prev.yarn.override { nodejs = final.pkgs.nodejs-16_x; };
      });

      #
      pkgs =
        import nixpkgs {
          inherit system;
          # Add your overlays to the list below. Note that they will be applied in order
          overlays = [ overlay ];
        };

    in rec {
      # For our outputs, we'll return the modified Yarn package from our overridden nixpkgs.
      packages = {
        yarn = pkgs.yarn;
      };
    }
  );
}

