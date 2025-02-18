# mariadb

## mariadb Notes

1. Start the mariadb server using `devbox services up`
1. Create a database using `"mysql -u root < setup_db.sql"`
1. You can now connect to the database from the command line by running `devbox run connect_db`

## Services

* mariadb

Use `devbox services start|stop [service]` to interact with services

## This plugin sets the following environment variables

* MYSQL_BASEDIR=/<projectDir>/.devbox/nix/profile/default
* MYSQL_HOME=/<projectDir>/.devbox/virtenv/mariadb/run
* MYSQL_DATADIR=/<projectDir>/.devbox/virtenv/mariadb/data
* MYSQL_UNIX_PORT=/<projectDir>/.devbox/virtenv/mariadb/run/mysql.sock
* MYSQL_PID_FILE=/<projectDir>/.devbox/virtenv/mariadb/run/mysql.pid

To show this information, run `devbox info mariadb`

Note that the `.sock` filepath can only be maximum 100 characters long. You can point to a different path by setting the `MYSQL_UNIX_PORT` env variable in your `devbox.json`.
