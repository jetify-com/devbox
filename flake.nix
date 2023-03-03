{
  description = "Devbox: Reproducible Dev Environments";
  inputs = {
    nixpkgs.url =
      "https://github.com/nixos/nixpkgs/archive/3954218cf613eba8e0dcefa9abe337d26bc48fd0.tar.gz";

    flake-utils.url = "github:numtide/flake-utils";

  };

  outputs = { self, nixpkgs, flake-utils }:
    let

      # to work with older version of flakes
      lastModifiedDate =
        self.lastModifiedDate or self.lastModified or "19700101";
      # System types to support.
      supportedSystems =
        [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });

      commit = builtins.substring 0 7 self.shortRev;
      version = "0.4.2";
    in {

      # Provide some binary packages for selected system types.
      packages = forAllSystems (system:
        let 
          pkgs = nixpkgsFor.${system};
          pname = "devbox";
          name = "devbox-${version}";

        in {
          devbox = pkgs.buildGoModule {
            inherit pname;
            inherit name;

            version = "${version}.${commit}";

            src = ./.;

            # integration tests want filesystem access
            doCheck = false;

            ldflags = [
              "-s"
              "-w"
              "-X go.jetpack.io/devbox/internal/build.Version=${version}"
            ];

            nativeBuildInputs = with pkgs; [ installShellFiles ];

            vendorSha256 =
              "sha256-62cJVlrGdrBSK+yzOA4WiHvplEMuKo09qp95+aX3WY0=";

            postInstall = ''
              installShellCompletion --cmd devbox \
                --bash <($out/bin/devbox completion bash) \
                --fish <($out/bin/devbox completion fish) \
                --zsh <($out/bin/devbox completion zsh)
            '';
          };
        });

      # Add dependencies that are only needed for development
      devShells = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go_1_19
              gopls
              gotools
              go-tools
              golangci-lint
            ];
          };
        });

      # The default package for 'nix build'. This makes sense if the
      # flake provides only one package or there is a clear "main"
      # package.
      defaultPackage = forAllSystems (system: self.packages.${system}.devbox);
    };
}
