---
title: PHP
---

PHP projects can manage most of their dependencies locally with `composer`. Some PHP extensions, however, need to be bundled with PHP at compile time. 

## Adding PHP to your Project

Run `devbox add php php80Packages.composer`, or add the following to your `devbox.json`:

```json
    "packages": [
        "php",
        "php81Packages.composer
    ]
```
This will install PHP 8.1 in your shell. 

Other versions available include: 

* `php80` (PHP 8.0)
* `php82` (PHP 8.2)

## Installing PHP Extensions

You can compile additional extensions into PHP by adding them to `packages` in your `devbox.json`. Devbox will automatically ensure that your extensions are included in PHP at compile time. 

For example -- to add the `ds` extension, run `devbox add php81Extensions.ds`, or update your packages to include the following: 

```json
    "packages": [
        "php",
        "php81Packages.composer",
        "php81Extensions.ds"
    ]
```

## Configuring PHP-FPM

PHP-FPM is a FastCGI Process Manager for PHP. It is automatically installed when you add `php` to your `devbox.json`, but requires some additional configuration to work with your local project:

1. First, you'll need to create a local `php-fpm.conf` file. 
2. Next, you will need to create an alias for `php-fpm` that points to the local conf file

### Create a local `php-fpm.conf`
Save the following conf somewhere in your project directory

```conf
[global]
pid = "${PHP_CONFDIR}"/php-fpm.pid
error_log = "${PHP_CONFDIR}/php-fpm.log"
daemonize = no

[www]
; user = www-data
; group = www-data
listen = "localhost:8081"
; listen.owner = www-data
; listen.group = www-data
pm = dynamic
pm.max_children = 5
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 3
chdir = /
```

### Point php-fpm to your local configuration

In your `init_hook`, set an environment variable to point to your local config.

```json
"init_hook": [
    "export PHP_CONF=<path_to_conf>",
    "export PHP_PORT=8080",
]
```
you can then run php-fpm using the local config using:

```bash
php-fpm -y $PHP_CONF -p $PWD
```