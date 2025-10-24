{
  description = "A flake that outputs PHP with custom extensions. Used by the devbox php plugin";

  inputs = {
    nixpkgs.url = "{{ .URLForInput }}";
  };

  outputs = { self, nixpkgs }:
    let
      extensions = builtins.concatLists(builtins.filter (x: x != null) [
        {{- range .Packages }}
        (builtins.match "^php.*Extensions\.([^@]*).*$" "{{ . }}")
        {{- end }}
      ]);

      php = (nixpkgs.legacyPackages.{{ .System }}.{{ .PackageAttributePath }}.withExtensions (
        { enabled, all }: enabled ++ (with all;
          map (ext: all.${ext}) extensions
        )
      )).overrideAttrs (oldAttrs: {
        # Skip tests in PHP and extensions to avoid flaky upstream tests
        # See: https://github.com/NixOS/nixpkgs/issues/...
        doCheck = false;
        doInstallCheck = false;
      });
    in
    {
      packages.{{ .System }} = {
        default = php;
        composer = php.packages.composer;
      };
    };
}
