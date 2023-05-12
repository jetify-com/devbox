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
    in
    {
      packages.aarch64-darwin = {    
        default = nixpkgs.legacyPackages.aarch64-darwin.php.withExtensions (
          { enabled, all }: enabled ++ (with all; 
            map (ext: all.${ext}) extensions
          )
        );
      };
    };
}
