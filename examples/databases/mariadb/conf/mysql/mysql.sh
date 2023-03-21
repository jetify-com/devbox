#! /bin/bash
# Installs the database if one doesn't exist, and then starts the mysql daemon. 
    set -eux
    alias mysql='mysql -u root'

    if [ ! -d "$MYSQL_DATADIR" ]; then
    # Install the Database
       mysql_install_db --auth-root-authentication-method=normal \
         --datadir=$MYSQL_DATADIR --basedir=$MYSQL_BASEDIR \
         --pid-file=$MYSQL_PID_FILE
    fi

    # Starts the daemon
    mysqld --datadir=$MYSQL_DATADIR --pid-file=$MYSQL_PID_FILE \
	    --socket=$MYSQL_UNIX_PORT 2> $MYSQL_HOME/mysql.log & MYSQL_PID=$!
    