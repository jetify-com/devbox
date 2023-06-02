---
title: PostgreSQL
---
PostgreSQL can be automatically configured by Devbox via the built-in Postgres Plugin. This plugin will activate automatically when you install Postgres using `devbox add postgresql`

[**Example Repo**](https://github.com/jetpack-io/devbox/tree/main/examples/databases/postgres)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/new?template=postgres)

## Adding Postgres to your Shell

`devbox add postgresql glibcLocales`, or in your `devbox.json`, add

```json
    "packages": [
        "postgresql@latest",
        "glibcLocales@latest"
    ]
```

This will install the latest version of Postgres. You can find other installable versions of Postgres by running `devbox search postgresql`.

## PostgreSQL Plugin Support

Devbox will automatically create the following configuration when you run `devbox add postgresql`:

### Services
* postgresql

You can use `devbox services start|stop postgresql` to start or stop the Postgres server in the background.

### Environment Variables

`PGHOST=./.devbox/virtenv/postgresql`
`PGDATA=./.devbox/virtenv/postgresql/data`

This variable tells PostgreSQL which directory to use for creating and storing databases.

### Notes

To initialize PostgreSQL run `initdb`. You also need to create a database using `createdb <db-name>`

