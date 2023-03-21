# postgresql-14.6

## postgresql Notes

You need to initialize and create a database as part of your setup.

1. Initialize a DB by running `initdb`
2. Start the Postgres server using `devbox services start`
3. Create a database using `createdb <name_of_db>`
4. You can now connect to the database from the command line by running `psql <name_of_db>`

To start the database manually run `pg_ctl -l .devbox/conf/postgresql/logfile start`.
To stop use `pg_ctl stop`.

## Services

* postgresql

Use `devbox services start|stop [service]` to interact with services

## This configuration sets the following environment variables

* PGDATA=/Users/johnlago/src/devbox-examples/databases/postgres/.devbox/conf/postgresql/data
* PGHOST=/Users/johnlago/src/devbox-examples/databases/postgres/.devbox/virtenv/postgresql

To show this information, run `devbox info postgresql`