{
  inputs = {
    nixpkgs-latest.url = "github:nixos/nixpkgs/3364b5b117f65fe1ce65a3cdd5612a078a3b31e3";
  };

  outputs = { self, nixpkgs-latest}:
    let 
      system = "x86_64-darwin";
      pkgs-latest = (import nixpkgs-latest {
        inherit system;
        config.allowUnfree = true;
      });
    in {
      packages.${system}.default = pkgs-latest.hello;
    };
}
