---
title: Using Devbox Plugins
---

This doc describes how to use Devbox Plugins with your project. **Plugins**  are provide a default Devbox configuration for a Nix package. Plugins make it easier to get started with packages that require additional setup when installed with Nix, and they offer a familiar interface for configuring packages. They also help keep all of your project's configuration within your project directory, which helps maintain portability and isolation.

If a plugin is available for your package, it will activate when you install the plugin using `devbox add <package name>`.  

## Current Plugins
Plugins are available for the following packages. You can activate the plugins for these packages by running `devbox add <package_name>`:
* [Apache](../devbox_examples/servers/apache.md) (apacheHttpd)
* [Nginx](../devbox_examples/servers/nginx.md) (nginx)
* [Caddy](../devbox_examples/servers/caddy.md) (caddy)
* [PostgreSQL](../devbox_examples/databases/postgres.md) (postgresql)
* [Redis](../devbox_examples/databases/redis.md) (redis)
* [PHP](../devbox_examples/languages/php.md) (php, php80, php81, php82)
* [Ruby](../devbox_examples/languages/ruby.md)(ruby, ruby_3_1, ruby_3_0)
* [Pip](../devbox_examples/languages/python.md) (python39Packages.pip, python310Packages.pip, python311Packages.pip)

Our team is rapidly adding new plugins to Devbox. If you want to request a plugin, please file an issue in the Devbox Repo.

## An Example of a Plugin: Nginx
Let's take a look at the plugin for Nginx. To get started, let's initialize a new devbox project, and add the `nginx` package:

```bash
cd ~/my_proj
devbox init && devbox add nginx
```

Devbox will install the package, activate the `nginx` plugin, and print a short explanation of the plugin's configuration

```bash
Installing nix packages. This may take a while... done.

nginx NOTES:
nginx can be configured with env variables

To customize:
* Use $NGINX_CONFDIR to change the configuration directory
* Use $NGINX_LOGDIR to change the log directory
* Use $NGINX_PIDDIR to change the pid directory
* Use $NGINX_RUNDIR to change the run directory
* Use $NGINX_SITESDIR to change the sites directory
* Use $NGINX_TMPDIR to change the tmp directory. Use $NGINX_USER to change the user
* Use $NGINX_GROUP to customize.

Services:
* nginx

Use `devbox services start|stop [service]` to interact with services

This plugin creates the following helper files:
* ~/my_project/devbox.d/nginx/nginx.conf
* ~/my_project/devbox.d/nginx/fastcgi.conf
* ~/my_project/devbox.d/web/index.html

This plugin sets the following environment variables:
* NGINX_CONFDIR=~/my_project/devbox.d/nginx/nginx.conf
* NGINX_PATH_PREFIX=~/my_project/.devbox/virtenv/nginx
* NGINX_TMPDIR=~/my_project/.devbox/virtenv/nginx/temp

To show this information, run `devbox info nginx`

nginx (nginx-1.22.1) is now installed.
```

Based on this info page, we can see that Devbox has created the configuration we need to run `nginx` in our local shell. Let's take a look at the files it created: 

```bash
% tree
.
├── devbox.d
│   ├── nginx
│   │   ├── fastcgi.conf
│   │   └── nginx.conf
│   └── web
│       └── index.html
└── devbox.json
```

These files give us everything we need to run NGINX, and we can modify the `nginx.conf` and `fastcgi.conf` to customize how Nginx works. 

We can also see in the info page that Devbox has configured an NGINX service for us. Let's start this service with `devbox services start nginx`, and then test it with `curl`:

```bash
> devbox services start nginx

Installing nix packages. This may take a while... done.
Starting a devbox shell...
Service "nginx" started

> curl localhost:80 
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>Hello World!</title>
  </head>
  <body>
    Hello World!
  </body>
</html>
```

## Plugin Configuration in detail

When Devbox detects a plugin for an installed package, it automatically applies its configuration and prints a short explanation. Developers can review this explanation anytime using `devbox info <package_name>`. 

### Services

If your package can run as a daemon or background service, Devbox can configure and manage that service for you with `devbox services`.

To learn more, visit our page on Devbox Services

### Environment Variables 
Devbox stores default environment variables for your package in `.devbox/virtenv/<package_name>/.env` in your project directory. Devbox automatically updates these environment variables whenever you run `devbox shell` or `devbox run` to match your current project, and developers should not check these `.env` files into source control.

#### Customizing Environment Variables
If you want to customize the environment variables, you can override them in the `init_hook` of your `devbox.json`

### Helper Files
Helper files are files that your package may use for configuration purposes, such as NGINX's `nginx.conf` file. When installing a package, Devbox will check for helper files in your project's `devbox.d` folder and create them if they do not exist. If helper files are already present, Devbox will not overwrite them.

#### Customizing Helper Files
Developers should directly edit helper files and check them into source control if needed

## Plugins Source Code

Devbox Plugins are written in JSON and stored in the main Devbox Repo. You can view the source code of the current plugins [here](https://github.com/jetpack-io/devbox/tree/main/plugins)

