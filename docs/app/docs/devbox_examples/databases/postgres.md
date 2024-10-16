---
title: PostgreSQL
---
PostgreSQL can be automatically configured by Devbox via the built-in Postgres Plugin. This plugin will activate automatically when you install Postgres using `devbox add postgresql`

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/databases/postgres)

[![Open In Devspace](../../../static/img/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/postgres)

## Adding Postgres to your Shell

You can install the PostgreSQL server and client by running`devbox add postgresql`. In some Linux distributions, you may also need to add `glibcLocales`, which can be added using `devbox add glibcLocales --platform=x86_64-linux,aarch64-linux`.

Alternatively, you can add the following to your devbox.json:

```json
  "packages": {
    "postgresql": "latest",
    "glibcLocales": {
      "version":   "latest",
      "platforms": ["x86_64-linux", "aarch64-linux"]
    }
  }
```

This will install the latest version of Postgres. You can find other installable versions of Postgres by running `devbox search postgresql`. You can also view the available versions on [Nixhub](https://www.nixhub.io/packages/postgresql)

## PostgreSQL Plugin Support

Devbox will automatically create the following configuration when you run `devbox add postgresql`:

### Services

* postgresql

You can use `devbox services start|stop postgresql` to start or stop the Postgres server in the background.

### Environment Variables

`PGHOST=./.devbox/virtenv/postgresql`
`PGDATA=./.devbox/virtenv/postgresql/data`

This variable tells PostgreSQL which directory to use for creating and storing databases.

### NOTES

1. To initialize PostgreSQL run:

```sh
initdb
```

1. You also need to create a user using:

```sh
createuser --interactive
```

1. (OPTIONAL) If the user has no permissions to create or drop a database, you also need to create a database using:

```sh
createdb <db-name>
```

#### Using the `createuser` Command

Run the createuser command with the `-s` or `--superuser` option to create a superuser. This grants the user the ability to bypass all access permission checks within the database, effectively granting them extensive privileges including the ability to create and drop databases. Additionally, you can use the `-r` or `--createrole` option to allow the user to create new roles. Here's an example command:

```sh
createuser -s -r your_new_user_name
```

Replace `your_new_user_name` with the desired username for the new superuser.

Remember: Creating a superuser grants them significant power over the database system, so it should be done cautiously and only when absolutely necessary due to the potential security implications.

### Disabling the Postgres Plugin

You can disable the Postgres plugin by running `devbox add postgresql --disable-plugin`, or by setting the `disable_plugin` field to `true` in your package definition:

```json
{
    "packages": {
        "postgresql": {
            "version": "latest",
            "disable_plugin": true
        }
    }
}
```
