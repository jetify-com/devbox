---
title: MySQL
---
MySQL can be automatically configured for your dev environment by Devbox via the built-in MySQL Plugin. This plugin will activate automatically when you install MySQL using `devbox add mysql80` or `devbox add mysql57`.

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/databases/mysql)

[![Open In Devspace](../../../static/img/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/mysql)

## Adding MySQL to your Shell

`devbox add mysql80`, or in your `devbox.json` add

```json
    "packages": [
        "mysql80@latest"
    ]
```

You can also install Mysql 5.7 by using `devbox add mysql57`

You can manually add the MySQL Plugin to your `devbox.json` by adding it to your `include` list:

```json
    "include": [
        "plugin:mysql"
    ]
```

This will install the latest version of MySQL. You can find other installable versions of MySQL by running `devbox search mysql80` or `devbox search mysql57`. You can also view the available versions on [Nixhub](https://www.nixhub.io/packages/mysql80)

## MySQL Plugin Support

Devbox will automatically create the following configuration when you run `devbox add mysql80` or `devbox add mysql57`. You can view the full configuration by running `devbox info mysql`

### Services

* mysql

You can use `devbox services up|stop mysql` to start or stop the MySQL Server.

### Environment Variables

```bash
MYSQL_BASEDIR=.devbox/nix/profile/default
MYSQL_HOME=./.devbox/virtenv/mysql/run
MYSQL_DATADIR=./.devbox/virtenv/mysql/data
MYSQL_UNIX_PORT=./.devbox/virtenv/mysql/run/mysql.sock
MYSQL_PID_FILE=./.devbox/virtenv/mysql/run/mysql.pid
MYSQL_CONF=./devbox.d/mysql/my.cnf
```

### Files

The following helper file will be created in your project directory:

* \{PROJECT_DIR\}/devbox.d/mysql/my.cnf

The plugin will also create the following helper files in your project's `.devbox/virtenv` folder:

* mysql/flake.nix
* mysql/setup_db.sh
* mysql/process-compose.yaml

These files are used to setup your database and service, and should not be modified

### Notes

* This plugin wraps mysqld to work in your local project. For more information, see the `flake.nix` created in your `.devbox/virtenv/mysql` folder.
* This plugin will create a new database for your project in `MYSQL_DATADIR` if one doesn't exist on shell init.
* You can use `mysqld` to manually start the server, and `mysqladmin -u root shutdown` to manually stop it
* `.sock` filepath can only be maximum 100 characters long. You can point to a different path by setting the `MYSQL_UNIX_PORT` env variable in your `devbox.json` as follows:

```json
"env": {
    "MYSQL_UNIX_PORT": "/<some-other-path>/mysql.sock"
}
```

### Disabling the MySQL PLugin

You can disable the built-in MySQL plugin using `devbox add mysql80 --disable-plugin`, or by setting the `disable_plugin` field to `true` in your package definition:

```json
{
    "packages": {
        "mysql80": {
            "version": "latest",
            "disable_plugin": true
        }
    }
}
```
