#! bash
export MYSQL_BASEDIR=$(which mariadb | sed -r "s/\/bin\/mariadb//g")
export MYSQL_HOME=$PWD/devbox.d/mysql
export MYSQL_DATADIR=$MYSQL_HOME/data
export MYSQL_UNIX_PORT=$MYSQL_HOME/mysql.sock
export MYSQL_PID_FILE=$MYSQL_HOME/mysql.pid