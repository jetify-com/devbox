---
title: MariaDB
---
To use a local MariaDB for development with Devbox and Nix, you will need to configure MariaDB to install and manage the configuration locally. This can be done using Environment variables to create the data directory, pidfile, and unix sock locally.

[**Example Repo**](https://github.com/jetpack-io/devbox/tree/main/examples/databases/mariadb)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/github.com/jetpack-io/devbox?folder=examples/databases/mariadb)

## Adding MariaDB to your Shell

`devbox add mariadb`, or in your `devbox.json` add

```json
    "packages": [
        "mariadb"
    ]
```

## Environment Variables

This script should be sourced in the `init_hook` of our devbox.json. This sets environment variables needed to configure and start MariaDB, as well as setting the data directory to a local folder. 

```bash
# Environment variables for running MariaDB with local configuration
export MYSQL_BASEDIR=$(which mariadb | sed -r "s/\/bin\/mariadb//g")
export MYSQL_HOME=$PWD/conf/mysql # or another folder in your project directory
# Store DB data in a local folder
export MYSQL_DATADIR=$MYSQL_HOME/data
# Keep the socket and pidfile in our conf folder, and out of the Nix Store
export MYSQL_UNIX_PORT=$MYSQL_HOME/mysql.sock
export MYSQL_PID_FILE=$MYSQL_HOME/mysql.pid
```

## Installing the Database
We can check if the MySQL data directory exists, and if not install the database using the environment variables we configured:

```bash
    if [ ! -d "$MYSQL_DATADIR" ]; then
    # Install the Database
       mysql_install_db --auth-root-authentication-method=normal \
         --datadir=$MYSQL_DATADIR --basedir=$MYSQL_BASEDIR \
         --pid-file=$MYSQL_PID_FILE
    fi
```

## Starting the Daemon
Similarly, we can start the database using mysqld, passing the environment variables where needed

```bash
    mysqld --datadir=$MYSQL_DATADIR --pid-file=$MYSQL_PID_FILE \
	    --socket=$MYSQL_UNIX_PORT 2> $MYSQL_HOME/mysql.log & MYSQL_PID=$!
```

The daemon can be terminated using `mysqladmin`: 

```bash
  mysqladmin -u root --socket=$MYSQL_UNIX_PORT shutdown
```