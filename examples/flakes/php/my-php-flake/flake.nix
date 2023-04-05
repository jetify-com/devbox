{
  description = "A flake to install PHP 8.2 with memcached and ds extension";

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
          php = pkgs.php.withExtensions ({ enabled, all }: enabled ++ (with all; [ ds memcached ]));
          hello = pkgs.hello;
        };
      });
}
