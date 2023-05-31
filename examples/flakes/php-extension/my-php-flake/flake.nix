{
  description =
    "A flake that outputs PHP with a custom extension (skeleton.so) linked.";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        # Customize and export the PHP package with some extra configuration
        php-ext = pkgs.php.buildEnv {
            # extraConfig will add the line below to the php.ini in our Nix store.
            # ${self} is a variable representing the current flake
            extraConfig = ''
              extension=${self}/skeleton.so
            '';
        };
      in {
        packages = {
          # Export the PHP package with our custom extension as the default
          default = php-ext;
        };
      });
}
