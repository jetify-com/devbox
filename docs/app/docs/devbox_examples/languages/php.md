---
title: PHP
---

PHP projects can manage most of their dependencies locally with `composer`. Some PHP extensions, however, need to be bundled with PHP at compile time. 

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/development/php/php8.1)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/github.com/jetpack-io/devbox-examples?folder=development/php/php8.1)

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

## PHP Plugin Details

The PHP Plugin will provide the following configuration when you install a PHP runtime with `devbox add`

### Services
* php-fpm

Use `devbox services start|stop php-fpm` to start PHP-FPM in the background.

### Helper Files

* {PROJECT_DIR}/devbox.d/php81/php-fpm.conf

You can modify this file to configure your PHP-FPM server

### Environment Variables

```bash
PHPFPM_PORT=8082
PHPFPM_ERROR_LOG_FILE={PROJECT_DIR}/.devbox/virtenv/php81/php-fpm.log
PHPFPM_PID_FILE={PROJECT_DIR}/.devbox/virtenv/php81/php-fpm.log
```