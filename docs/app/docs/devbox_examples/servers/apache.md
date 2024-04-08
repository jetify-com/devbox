---
title: Apache
---

Apache can be automatically configured by Devbox via the built-in Apache Plugin. This plugin will activate automatically when you install Apache using `devbox add apache`.

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/servers/apache)

[![Open In Devbox.sh](https://jetify.com/img/devbox/open-in-devbox.svg)](https://devbox.sh/open/templates/apache)

### Adding Apache to your Shell

Run `devbox add apache`, or add the following to your `devbox.json`

```json
  "packages": [
    "apache@latest"
  ]
```

This will install the latest version of Apache. You can find other installable versions of Apache by running `devbox search apache`. You can also view the available versions on [Nixhub](https://www.nixhub.io/packages/apache)

## Apache Plugin Details

The Apache plugin will automatically create the following configuration when you install Apache with `devbox add`.

### Services
* apache

Use `devbox services start|stop apache` to start and stop httpd in the background.

### Helper Files
The following helper files will be created in your project directory:

* \{PROJECT_DIR\}/devbox.d/apache/httpd.conf
* \{PROJECT_DIR\}/devbox.d/web/index.html

Note that by default, Apache is configured with `./devbox.d/web` as the DocumentRoot. To change this, you should copy and modify the default `./devbox.d/apache/httpd.conf`.

### Environment Variables
```bash
HTTPD_ACCESS_LOG_FILE={PROJECT_DIR}/.devbox/virtenv/apache/access.log
HTTPD_ERROR_LOG_FILE={PROJECT_DIR}/.devbox/virtenv/apache/error.log
HTTPD_PORT=8080
HTTPD_DEVBOX_CONFIG_DIR={PROJECT_DIR}
HTTPD_CONFDIR={PROJECT_DIR}/devbox.d/apache
```

### Notes

We recommend copying your `httpd.conf` file to a new directory and updating HTTPD_CONFDIR if you decide to modify it.
