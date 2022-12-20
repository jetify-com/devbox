---
title: PostgreSQL
---
PostgreSQL can be automatically configured by Devbox via the built-in Postgres PLugin. This plugin will activate automatically when you install Postgres using `devbox add postgresql`

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/databases/postgres)

## Adding Postgres to your Shell

`devbox add postgresql glibcLocales`, or in your `devbox.json`, add

```json
    "packages": [
        "postgresql",
        "glibcLocales"
    ]
```

## PostgreSQL Plugin Support

Devbox will automatically create the following configuration when you run `devbox add postgresql`:

### Services
* postgresql

You can use `devbox services start|stop postgresql` to start or stop the Postgres server in the background.

### Environment Variables

`PGDATA=./.devbox/virtenv/postgresql/data`

This variable tells PostgreSQL which directory to use for creating and storing databases. 

### Notes

To initialize PostgreSQL run `initdb`. You also need to create a database using `createdb <db-name>`

