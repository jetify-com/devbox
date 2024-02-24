{
  description =
    "This flake builds the Ory CLI using Nix's buildGoModule Function.";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    # The Ory CLI is not a flake, so we have to use the Github input and build it ourselves.
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
      # Define some variables that we want to use in our package build. You'll want to update version and `ref` above to use a different version of Ory.
      version = "0.2.2";
    in {
      packages = let
          pkgs = import nixpkgs{inherit system;};
          pname = "ory";
          name = "ory-${version}";
      in {
        # Build the Ory CLI using Nix's buildGoModuleFunction
        ory-cli = pkgs.buildGoModule {
          inherit version;
          inherit pname;
          inherit name;

          # Path to the source code we want to build. In this case, it's the `ory-cli` input we defined above.
          src = ory-cli;

          # This was in the Makefile in the Ory repo, not sure if it's required
          tags = [ "sqlite"];

          doCheck = false;

          # If the vendor folder is not checked in, we have to provide a hash for the vendor folder. Nix requires this to ensure the vendor folder is reproducible, and matches what we expect.
          vendorSha256 = "sha256-J9jyeLIT+1pFnHOUHrzmblVCJikvY05Sw9zMz5qaDOk=";

          # The Go Mod is named `cli` by default, so we rename it to `ory`.
          postInstall = ''
            mv $out/bin/cli $out/bin/ory
          '';
        };
      };

      # Set Ory as the default package output for this flake
      defaultPackage = self.packages.ory-cli;
    }
  );
}
