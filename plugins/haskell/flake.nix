{
  description = "A flake that outputs haskell with custom packages. Used by the devbox haskell plugin";

  inputs = {
    nixpkgs.url = "{{ .URLForInput }}";
  };

  outputs = { self, nixpkgs }:
    let
      version = builtins.elemAt (builtins.match "^haskell\.compiler\.(.*)$" "{{ .PackageAttributePath }}") 0;
      
      ghcWithPackages = if "{{ .PackageAttributePath }}" == "ghc" 
        then nixpkgs.legacyPackages.{{ .System }}.pkgs.haskellPackages.ghcWithPackages
        else nixpkgs.legacyPackages.{{ .System }}.pkgs.haskell.packages.${version}.ghcWithPackages;

      haskellPackages = builtins.concatLists(builtins.filter (x: x != null) [
        {{- range .Packages }}
        # Test if {{ . }} is a haskell package
        (builtins.match "^(stack|cabal-install)$" "{{ . }}")
        (builtins.match "^haskellPackages\.(.*)$" "{{ . }}")
        (builtins.match "^haskell\.packages\.[^.]*\.(.*)$" "{{ . }}")
        {{- end }}
      ]);
    in
    {
      packages.{{ .System }} = {
        default = ghcWithPackages (ps: with ps;
          map (haskellPackage: ps.${haskellPackage}) haskellPackages
        );
      };
    };
}
