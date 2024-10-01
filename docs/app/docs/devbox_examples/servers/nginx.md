---
title: Nginx
---

NGINX can be automatically configured by Devbox via the built-in NGINX Plugin. This plugin will activate automatically when you install NGINX using `devbox add nginx`

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/servers/nginx)

[![Open In Devspace](../../../static/img/open-in-devspace.svg)](https://www.jetify.com/devbox/templates/nginx)

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
* devbox.d/nginx/nginx.template
* devbox.d/nginx/fastcgi.conf
* devbox.d/web/index.html

Devbox uses [envsubst](https://www.gnu.org/software/gettext/manual/html_node/envsubst-Invocation.html) to generate `nginx.conf` from the `nginx.template` file every time Devbox starts a shell, service, or script. This allows you to create an NGINX config using environment variables by modifying `nginx.template`. To edit your NGINX configuration, you should modify the `nginx.template` file.

Note that by default, NGINX is configured with `./devbox.d/web` as the root directory. To change this, you should modify `./devbox.d/nginx/nginx.template`

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

### Disabling the NGINX Plugin

You can disable the NGINX plugin by running `devbox add nginx --disable-plugin`, or by setting the `disable_plugin` field in your `devbox.json`:

```json
{
  "packages": {
    "nginx": {
      "version": "latest",
      "disable_plugin": true
    }
  }
}
```
