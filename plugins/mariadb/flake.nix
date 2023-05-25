{
  description = "A flake that outputs MariaDB with custom configuration and aliases to work in Devbox";

  inputs = {
    nixpkgs.url = "{{.URLForInput}}";
  };

  outputs = {self, nixpkgs}:
    let
      mariadb-bin =  nixpkgs.legacyPackages.{{.System}}.symlinkJoin {

        name = "mariadb-wrapped";
        paths = [nixpkgs.legacyPackages.{{ .System }}.mariadb];
        nativeBuildInputs = [ nixpkgs.legacyPackages.{{.System}}.makeWrapper ];
        postBuild = ''
          wrapProgram "$out/bin/mysql" \
            --add-flags '--socket=''$MYSQL_UNIX_PORT';

          wrapProgram $out/bin/mysqladmin \
            --add-flags '--socket=''$MYSQL_UNIX_PORT';

          wrapProgram $out/bin/mysqld \
            --add-flags '--datadir=''$MYSQL_DATADIR --pid-file=''$MYSQL_PID_FILE --socket=''$MYSQL_UNIX_PORT';

          wrapProgram "$out/bin/mysql_install_db" \
            --add-flags '--datadir=''$MYSQL_DATADIR --pid-file=''$MYSQL_PID_FILE --basedir=''$MYSQL_BASEDIR';

        '';
      };
    in{
      packages.{{.System}} = {
        default = mariadb-bin;
      };
    };
}
