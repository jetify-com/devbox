# Nginx

NGINX can be automatically configured by Devbox via the built-in NGINX Plugin. This plugin will activate automatically when you install NGINX using `devbox add nginx`

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/servers/nginx)

[![Open In Devspace](https://www.jetify.com/img/devbox/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/nginx)

## Adding NGINX to your Shell

Run `devbox add nginx`, or add the following to your `devbox.json`

```json
  "packages": [
    "nginx@latest"
  ]
```

This will install the latest version of NGINX. You can find other installable versions of NGINX by running `devbox search nginx`. You can also view the available versions on [Nixhub](https://www.nixhub.io/packages/nginx)

## NGINX Plugin Details

### Services

* nginx

Use `devbox services start|stop nginx` to start and stop the NGINX service in the background

### Helper Files

The following helper files will be created in your project directory:

* devbox.d/nginx/nginx.conf
* devbox.d/nginx/fastcgi.conf
* devbox.d/web/index.html

Note that by default, NGINX is configured with `./devbox.d/web` as the root directory. To change this, you should modify `./devbox.d/nginx/nginx.conf`

### Environment Variables

```bash
NGINX_CONFDIR=devbox.d/nginx/nginx.conf
NGINX_PATH_PREFIX=.devbox/virtenv/nginx
NGINX_TMPDIR=.devbox/virtenv/nginx/temp
```

### Notes

You can easily configure NGINX by modifying these env variables in your shell's `init_hook`

To customize:

* Use $NGINX_CONFDIR to change the configuration directory
* Use $NGINX_LOGDIR to change the log directory
* Use $NGINX_PIDDIR to change the pid directory
* Use $NGINX_RUNDIR to change the run directory
* Use $NGINX_SITESDIR to change the sites directory
* Use $NGINX_TMPDIR to change the tmp directory. Use $NGINX_USER to change the user
* Use $NGINX_GROUP to customize.

You can also customize the `nginx.conf` and `fastcgi.conf` stored in `devbox.d/nginx`
