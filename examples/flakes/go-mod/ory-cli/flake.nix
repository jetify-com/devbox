{
  description =
    "This flake outputs a modified version of Yarn that uses NodeJS 16";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    ory-cli = {
      type = "github";
      owner = "ory";
      repo = "cli";
      ref = "v0.2.2";
      flake = false;
    };
  };

  outputs = { self, nixpkgs, flake-utils, ory-cli }:
  # Use the flake-utils lib to easily create a multi-system flake
  flake-utils.lib.eachDefaultSystem (system:
    let
      # You can define overlays as functions using the example below
      # This overlay will modify yarn to use nodejs-16_x
      version = "0.2.2";
    in {
      # For our outputs, we'll return the modified Yarn package from our overridden nixpkgs.
      packages = let
          pkgs = import nixpkgs{inherit system;};
          pname = "ory";
          name = "ory-${version}";
      in {
        ory-cli = pkgs.buildGoModule {
          inherit version;
          inherit pname;
          inherit name;

          src = ory-cli;

          ldFlags = ["-o=${pname}"];

          tags = [ "sqlite"];

          doCheck = false;

          vendorSha256 = "sha256-J9jyeLIT+1pFnHOUHrzmblVCJikvY05Sw9zMz5qaDOk=";

          postInstall = ''
            mv $out/bin/cli $out/bin/ory
          '';
        };
      };

      # [Optional] Set yarn as the default package output for this flake
      defaultPackage = self.packages.ory-cli;
    }
  );
}

