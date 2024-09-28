{
  description = "Instant, easy, predictable dev environments";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        lastTag = "0.13.2";

        # Add the commit to the version string, in case someone builds from main
        getVersion = pkgs.lib.trivial.pipe self [
          (x: "${lastTag}")
          (x: if (self ? shortRev)
              then "${x}-${self.shortRev}"
              else "${x}-${if self ? dirtyShortRev then self.dirtyShortRev else "dirty"}")
        ];

        # Run `devbox run update-flake` to update the vendorHash
        vendorHash = if builtins.pathExists ./vendor-hash
                     then builtins.readFile ./vendor-hash
                     else "";

        buildGoModule = pkgs.buildGo123Module;

      in
      {
        inherit self;
        packages.default = buildGoModule rec {
          pname = "devbox";
          version = getVersion;

          src = ./.;

          inherit vendorHash;

          ldflags = [
            "-s"
            "-w"
            "-X go.jetpack.io/devbox/internal/build.Version=${version}"
          ];

          # Disable tests if they require network access or are integration tests
          doCheck = false;

          nativeBuildInputs = [ pkgs.installShellFiles ];

          postInstall = ''
            installShellCompletion --cmd devbox \
              --bash <($out/bin/devbox completion bash) \
              --fish <($out/bin/devbox completion fish) \
              --zsh <($out/bin/devbox completion zsh)
          '';

          meta = with pkgs.lib; {
            description = "Instant, easy, and predictable development environments";
            homepage = "https://www.jetify.com/devbox";
            license = licenses.asl20;
            maintainers = with maintainers; [ lagoja ];
          };
        };
      }
    );
}
