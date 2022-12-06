---
title: Postgres
---
To use a local Postgres DB instance for Development with Devbox and Nix, you'll need to configure a local Data directory for Postgres to install your DB. This can be done by setting a PGDATA environment variable in the `init_hook` of your `devbox.json`

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/databases/postgres)

## Adding Postgres to your Shell

`devbox add postgresql glibcLocales`, or in your `devbox.json`, add

```json
    "packages": [
        "postgresql",
        "glibcLocales"
    ]
```

## Environment Variables

These environment variable should be sourced in the `init_hook` of your `devbox.json`. 

```json
"init_hook": [
    "export PG_CONFDIR=$PWD/conf/postgresql"
    "export PGDATA=$PG_CONFDIR/data"
]
```

## Installing the Database

If your PGDATA directory does not exist, we can create and install your DB there with the following script:

```bash
if [ ! -d $PGDATA ]; then
    pg_ctl init
fi
```

You can add this to your `init_hook`, or in a script in your `devbox.json`

