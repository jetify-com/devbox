#! bash

if [ ! -d "$MYSQL_DATADIR" ]; then
# Install the Database
   mkdir $MYSQL_DATADIR
   mysqld --initialize-insecure
fi
