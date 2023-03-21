#! bash

# This script should be sourced in the init_hook of our devbox.json
# This sets environment variables needed to configure and start MariaDB, as well as some of our Configuration Paths

# Environment variables for running MariaDB with local configuration
export MYSQL_BASEDIR=$(which mariadb | sed -r "s/\/bin\/mariadb//g")
export MYSQL_HOME=$PWD/conf/mysql
# Store DB data in a local folder
export MYSQL_DATADIR=$MYSQL_HOME/data
# Keep the socket and pidfile in our conf folder
export MYSQL_UNIX_PORT=$MYSQL_HOME/mysql.sock
export MYSQL_PID_FILE=$MYSQL_HOME/mysql.pid