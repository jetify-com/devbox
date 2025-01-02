#! bash

if [ ! -d "$MYSQL_DATADIR" ]; then
# Install the Database
    mysql_install_db --auth-root-authentication-method=normal \
        --datadir=$MYSQL_DATADIR --basedir=$MYSQL_BASEDIR \
        --pid-file=$MYSQL_PID_FILE
fi

if [ -e "$MYSQL_CONF" ]; then
  ln -fs "$MYSQL_CONF" "$MYSQL_HOME/my.cnf"
fi