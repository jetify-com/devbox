#! bash

if [ ! -d "$MYSQL_DATADIR" ]; then
  # Install the Database
  mkdir $MYSQL_DATADIR
  mysqld --initialize-insecure
fi

if [ -e "$MYSQL_CONF" ]; then
  ln -fs "$MYSQL_CONF" "$MYSQL_HOME/my.cnf"
fi