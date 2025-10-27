{
  description = "A flake that outputs MySQL with custom configuration and aliases to work in Devbox";

  inputs = {
    nixpkgs.url = "{{.URLForInput}}";
  };

  outputs = {self, nixpkgs}:
    let
      mysql-bin =  nixpkgs.legacyPackages.{{.System}}.symlinkJoin {

        name = "mysql-wrapped";
        paths = [nixpkgs.legacyPackages.{{ .System }}.{{.PackageAttributePath}}];
        nativeBuildInputs = [ nixpkgs.legacyPackages.{{.System}}.makeWrapper];
        postBuild = ''

          wrapProgram $out/bin/mysqld \
            --add-flags '--basedir=''$MYSQL_BASEDIR --datadir=''$MYSQL_DATADIR --pid-file=''$MYSQL_PID_FILE --socket=''$MYSQL_UNIX_PORT';

          wrapProgram $out/bin/mysqld_safe \
            --add-flags '--basedir=''$MYSQL_BASEDIR --datadir=''$MYSQL_DATADIR --pid-file=''$MYSQL_PID_FILE --socket=''$MYSQL_UNIX_PORT';
        '';
      };
    in{
      packages.{{.System}} = {
        default = mysql-bin;
      };
    };
}
