---
title: LAPP (Linux, Apache, PHP, Postgres)
---

This example shows how to build a simple application using Apache, PHP, and PostgreSQL. It uses Devbox Plugins for all 3 packages to simplify configuration

[Example Repo](https://github.com/jetify-com/devbox/tree/main/examples/stacks/lapp-stack)


## How to Run

1. To start Apache, PHP-FPM, and Postgres in the background, run `devbox service start`.
2. Once the services are running, you can start your shell using `devbox shell`. This will also initialize your database by running `initdb` in the init hook.
3. Create the database and load the test data by using `devbox run create_db`.
4. You can now test the app using `localhost:8080` to hit the Apache Server. If you want Apache to listen on a different port, you can change the `HTTPD_PORT` environment variable in the Devbox init_hook.

## How to Recreate this Example

1. Create a new project with `devbox init`
1. Add the packages using the command below. Installing the packages with `devbox add` will ensure that the plugins are activated:

```bash
devbox add postgresql@14 php php83Extensions.pgsql@latest apache@2.4
```

1. Update `devbox.d/apache/httpd.conf` to point to the directory with your PHP files. You'll need to update the `DocumentRoot` and `Directory` directives.
1. Follow the instructions above in the How to Run section to initialize your project
