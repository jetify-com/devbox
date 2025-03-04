{
  description = "A flake that outputs MariaDB with custom configuration and aliases to work in Devbox";

  inputs = {
    nixpkgs.url = "{{.URLForInput}}";
  };

  outputs = {self, nixpkgs}:
    let
      mariadb-bin =  nixpkgs.legacyPackages.{{.System}}.symlinkJoin {

        name = "mariadb-wrapped";
        paths = [nixpkgs.legacyPackages.{{ .System }}.{{.PackageAttributePath}}];
        nativeBuildInputs = [ nixpkgs.legacyPackages.{{.System}}.makeWrapper];
        postBuild = ''

          wrapProgram $out/bin/mysqld \
            --add-flags '--basedir=$out --datadir=''$MYSQL_DATADIR --pid-file=''$MYSQL_PID_FILE --socket=''$MYSQL_UNIX_PORT';

          wrapProgram $out/bin/mariadbd \
            --add-flags '--basedir=$out --datadir=''$MYSQL_DATADIR --pid-file=''$MYSQL_PID_FILE --socket=''$MYSQL_UNIX_PORT';

          wrapProgram $out/bin/mysqld_safe \
            --add-flags '--basedir=$out --datadir=''$MYSQL_DATADIR --pid-file=''$MYSQL_PID_FILE --socket=''$MYSQL_UNIX_PORT';

          if [-f $out/bin/mariadbd-safe]; then
            wrapProgram $out/bin/mariadbd_safe \
              --add-flags '--basedir=$out --datadir=''$MYSQL_DATADIR --pid-file=''$MYSQL_PID_FILE --socket=''$MYSQL_UNIX_PORT';
          fi

          wrapProgram "$out/bin/mysql_install_db" \
            --add-flags '--basedir=$out --datadir=''$MYSQL_DATADIR --pid-file=''$MYSQL_PID_FILE --basedir=''$MYSQL_BASEDIR';

          if [-f $out/bin/mariadb-install-db]; then
            wrapProgram "$out/bin/mariadb_install_db" \
              --add-flags '--basedir=$out --datadir=''$MYSQL_DATADIR --pid-file=''$MYSQL_PID_FILE --basedir=''$MYSQL_BASEDIR';
          fi
        '';
      };
    in{
      packages.{{.System}} = {
        default = mariadb-bin;
      };
    };
}
