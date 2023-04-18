{
  description =
    "A flake that outputs PHP with a custom extension (skeleton.so) linked.";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages = {
          # Customize and export the PHP package with some extra config
          php = pkgs.php.buildEnv {
            # extraConfig will add the line below to our php.ini
            # ${self} is a variable representing the current flake
            extraConfig = ''
              extension=${self}/skeleton.so 
            '';
          };
        };
      });
}