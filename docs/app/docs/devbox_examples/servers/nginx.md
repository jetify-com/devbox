---
title: Nginx
---

Nginx, when installed with Devbox and Nix, will by default attempt to store it's configuration (`nginx.conf`), Pidfile, and temporary files in the Nix Store. This will cause issues when trying to configure your project, since the Nix Store is immutable and should not be modified after installation.

To use Nginx with your project, you'll need to configure Nginx to use a local conf file and temporary directory

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/servers/nginx)

### Adding NGINX to your Shell

Run `devbox add nginx`, or add the following to your `devbox.json`

```json
  "packages": [
    "nginx"
  ]
```

### Environment Variables

To make it easy to setup NGINX, we can define a few Environment variables that will point to our local configuration and temp directories. Add the following to the `init_hook` in your `devbox.json`.

```json
"init_hook": [
    "export NGINX_CONFDIR=$PWD/conf/nginx"
]
```

### Adding a local `nginx.conf`

To configure your server, you'll need to add a local `nginx.conf` file to your project. Here is a simple one that can get you started, using the env variables in our init_hook

```conf
events {}
http{
server {
         listen       80;
         listen       [::]:80;
         server_name  localhost;
         root         ../../web;

         error_log  error.log error;
         access_log access.log;
         client_body_temp_path temp/client_body;
         proxy_temp_path temp/proxy;
         fastcgi_temp_path temp/fastcgi;
         uwsgi_temp_path temp/uwsgi;
         scgi_temp_path temp/scgi;

         index index.html;
         server_tokens off;
    }
}
```

### Using the configuration

When starting NGINX, we'll need to point it to our local configuration. We can do it with the following command: 

```bash
nginx -p $NGINX_CONFDIR -c nginx.conf -e error.log -g "pid nginx.pid"
```

To stop it, we'll also need to use a similar command: 

```bash
nginx -p $NGINX_CONFDIR -c nginx.conf -e error.log -g "pid nginx.pid;" -s stop
```

The [example repo](https://github.com/jetpack-io/devbox-examples/tree/main/servers/nginx) shows how you can use the `init_hook` to start NGINX automatically when you launch your shell, and stop it when the shell exits.
