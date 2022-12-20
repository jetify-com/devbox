---
title: Running Background Services
---

When working on an application, you often want some services or dependencies running in the background for testing. Take a web app as an example. While working on your application, you will want to test it against a running development server and database. Previously developers would manage these services via tools like Docker Compose or orchestrating them manually.

With Devbox, you can manage these services from the CLI using `devbox services`. 

:::note

Currently, Devbox Services are only available via [Plugins](plugins.md). Future releases of Devbox will make it possible to configure your own services in your `devbox.json`

:::

## Plugins that Support Services

The following plugins provide a service that can be managed with `devbox services`: 

* [Apache](../devbox_examples/servers/apache.md) (apacheHttpd)
* [Nginx](../devbox_examples/servers/nginx.md) (nginx)
* [PostgreSQL](../devbox_examples/databases/postgres.md) (postgresql)
* [PHP](../devbox_examples/languages/php.md) (php, php80, php81, php82)

The service will be made available to your project when you install the packages using `devbox add`. 

## Listing the Services in our Project

You can list all the services available to your current devbox project by running `devbox services ls`. For example, the services in a PHP web app project might look like this:

```bash
devbox services ls

php-fpm
apache
postgresql
```

## Starting your Services

You can start all the services in your project by running `devbox services start`:

```bash
devbox services start

Installing nix packages. This may take a while... done.
Starting a devbox shell...
Service "php-fpm" started
Service "apache" started
waiting for server to start.... done
server started
Service "postgresql" started
```

You can also start a specific service by passing the name as an argument. For example, to start just `postgresql`, you can run `devbox services start postgresql`

If you want to restart your services (for example, after changing your configuration), you can run `devbox services restart`

## Stopping your services

You can stop your services with `devbox services stop`. This will stop all the running services associated with your project: 

```bash
devbox services stop

Installing nix packages. This may take a while... done.
Starting a devbox shell...
Service "php-fpm" stopped
Service "apache" stopped
waiting for server to shut down.... done
server stopped
Service "postgresql" stopped
```

## Further Reading

* [**Devbox Services CLI Reference**](../cli_reference/devbox_services.md)
* [**Devbox Plugins**](plugins.md)