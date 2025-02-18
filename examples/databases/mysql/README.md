# mysql

## mysql Notes

1. Start the mysql server using `devbox services up`
1. Create a database using `"mysql -u root < setup_db.sql"`
1. You can now connect to the database from the command line by running `devbox run connect_db`

## Services

* mysql

Use `devbox services start|stop [service]` to interact with services

## This plugin sets the following environment variables

* MYSQL_BASEDIR=&lt;projectDir>/.devbox/nix/profile/default
* MYSQL_HOME=&lt;projectDir>/.devbox/virtenv/mysql/run
* MYSQL_DATADIR=&lt;projectDir>/.devbox/virtenv/mysql/data
* MYSQL_UNIX_PORT=&lt;projectDir>/.devbox/virtenv/mysql/run/mysql.sock
* MYSQL_PID_FILE=&lt;projectDir>/.devbox/virtenv/mysql/run/mysql.pid

To show this information, run `devbox info mysql`

Note that the `.sock` filepath can only be maximum 100 characters long. You can point to a different path by setting the `MYSQL_UNIX_PORT` env variable in your `devbox.json`.
