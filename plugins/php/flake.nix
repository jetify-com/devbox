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

      php = nixpkgs.legacyPackages.{{ .System }}.{{ .PackageAttributePath }}.withExtensions (
        { enabled, all }: enabled ++ (with all; 
          map (ext: all.${ext}) extensions
        )
      );
    in
    {
      packages.{{ .System }} = {
        # This flake exports more than one package (php and composer), so each
        # must be referenced from the plugin via an explicit url fragment
        # (e.g. path:.../flake#php). The bare `default` output is kept for
        # backwards compatibility with anyone referencing the flake directly.
        default = php;
        php = php;
        composer = php.packages.composer;
      };
    };
}
