#! bash

if [ ! -d "$MYSQL_DATADIR" ]; then
  # Install the Database
  mkdir -p $MYSQL_DATADIR
  mysqld --initialize-insecure --basedir=$MYSQL_BASEDIR
fi

# Create run directory for socket files if it doesn't exist
MYSQL_RUN_DIR="$(dirname $MYSQL_UNIX_PORT)"
if [ ! -d "$MYSQL_RUN_DIR" ]; then
  mkdir -p "$MYSQL_RUN_DIR"
fi

if [ -e "$MYSQL_CONF" ]; then
  ln -fs "$MYSQL_CONF" "$MYSQL_HOME/my.cnf"
fi
