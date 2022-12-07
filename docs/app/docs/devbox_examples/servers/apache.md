---
title: Apache
---

Apache, when installed with Devbox and Nix, will by default attempt to store it's configuration (`apache.conf`), Pidfile, and logs in the Nix Store. This will cause issues when trying to configure your project, since the Nix Store is immutable and can't be modified after installation

To use Apache with your project, you'll need to configure Apache to use a local conf file and temporary directory

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/servers/apache)

### Adding Apache to your Shell

Run `devbox add apacheHttpd`, or add the following to your `devbox.json`

```json
  "packages": [
    "apacheHttpd"
  ]
```

### Environment Variables

To make it easy to setup Apache, we can define a few Environment variables that will point to our local configuration and temp directories. Add the following to the `init_hook` in your `devbox.json`.

```json
"init_hook": [
    "export HTTPD_CONFDIR=$PWD/conf/httpd",
    "export HTTPD_PORT=80"
]
```

### Adding a local `apache.conf`

To configure your server, you'll need to add a local `apache.conf` file to your project. Here is a simple one that can get you started, using the env variables in our init_hook

```conf
ServerAdmin             "root@localhost"
ServerName              "devbox-apache"
Listen                  "${HTTPD_PORT}"
PidFile                 "${HTTPD_CONFDIR}/apache.pid"
UseCanonicalName        Off

LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule authz_host_module modules/mod_authz_host.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule unixd_module modules/mod_unixd.so
LoadModule dir_module modules/mod_dir.so

<IfModule unixd_module>
    User daemon
    Group daemon
</IfModule>

<Directory />
    AllowOverride none
    Require all denied
</Directory>

<Files ".ht*">
    Require all denied
</Files>

ErrorLog "${HTTPD_CONFDIR}/error.log"

<VirtualHost "*:${HTTPD_PORT}">

    DocumentRoot "${PWD}/web"

    <Directory "${PWD}/web">
        Options All
        AllowOverride All
        <IfModule mod_authz_host.c>
            Require all granted
        </IfModule>
    </Directory>

    DirectoryIndex index.html 

</VirtualHost>
```

### Using the configuration

When starting httpd, we'll need to point it to our local configuration. If we're using apachectl, we can do it with the following command: 

```bash
apachectl start -f $HTTPD_CONFDIR/httpd.conf"
```

To stop it, we'll also need to use a similar command: 

```bash
apachectl stop -f $HTTPD_CONFDIR/httpd.conf
```

The [example repo](https://github.com/jetpack-io/devbox-examples/tree/main/servers/apache) shows how you can use the `init_hook` to start apache automatically when you launch your shell, and stop it when the shell exits.
