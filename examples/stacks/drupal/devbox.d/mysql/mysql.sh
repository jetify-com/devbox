#! /bin/bash
    
    set -eux
    alias mysql='mysql -u root'

    if [ ! -d "$MYSQL_DATADIR" ]; then
      # Make sure to use normal authentication method otherwise we can only
      # connect with unix account. But users do not actually exists in nix.
       mysql_install_db --auth-root-authentication-method=normal \
         --datadir=$MYSQL_DATADIR --basedir=$MYSQL_BASEDIR \
         --pid-file=$MYSQL_PID_FILE
    fi

    # Starts the daemon
    mysqld --datadir=$MYSQL_DATADIR --pid-file=$MYSQL_PID_FILE \
	    --socket=$MYSQL_UNIX_PORT 2> $MYSQL_HOME/mysql.log & MYSQL_PID=$!
    