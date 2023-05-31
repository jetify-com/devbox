{
  description = "A flake that outputs PHP with memcached and ds extension and hello pkg.";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          # Flakes can export multiple packages. To include specific packages in
          # devbox.json you can use url fragments (e.g. path:my-flake#my-package)
          php = pkgs.php.withExtensions ({ enabled, all }: enabled ++ (with all; [ ds memcached ]));
          hello = pkgs.hello;

          # If you only want to export a single package, you can name it default which allows
          # installation without using url fragment (.e.g. "path:my-flake")
          # default = pkgs.php.withExtensions ({ enabled, all }: enabled ++ (with all; [ ds memcached ]));
        };
      });
}
