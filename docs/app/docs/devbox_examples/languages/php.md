---
title: PHP
---

PHP projects can manage most of their dependencies locally with `composer`. Some PHP extensions, however, need to be bundled with PHP at compile time.

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/development/php/latest)

[![Open In Devbox.sh](https://www.jetify.com/img/devbox/open-in-devbox.svg)](https://devbox.sh/open/templates/php)

## Adding PHP to your Project

Run `devbox add php php83Packages.composer`, or add the following to your `devbox.json`:

```json
    "packages": [
        "php@latest",
        "php83Packages.composer@latest
    ]
```

If you want a different version of PHP for your project, you can search for available versions by running `devbox search php`. You can also view the available versions on [Nixhub](https://www.nixhub.io/packages/php)

## Installing PHP Extensions

You can compile additional extensions into PHP by adding them to `packages` in your `devbox.json`. Devbox will automatically ensure that your extensions are included in PHP at compile time.

For example -- to add the `ds` extension, run `devbox add php83Extensions.ds`, or update your packages to include the following:

```json
    "packages": [
        "php@latest",
        "php83Packages.composer",
        "php83Extensions.ds"
    ]
```

## PHP Plugin Details

The PHP Plugin will provide the following configuration when you install a PHP runtime with `devbox add`. You can also manually add the PHP plugin by adding `plugin:php` to your `include` list in `devbox.json`:

```json
    "include": [
        "plugin:php"
    ]
```

### Services
* php-fpm

Use `devbox services start|stop php-fpm` to start PHP-FPM in the background.

### Environment Variables

```bash
PHPFPM_PORT=8082
PHPFPM_ERROR_LOG_FILE={PROJECT_DIR}/.devbox/virtenv/php/php-fpm.log
PHPFPM_PID_FILE={PROJECT_DIR}/.devbox/virtenv/php/php-fpm.pid
PHPRC={PROJECT_DIR}/devbox.d/php/php.ini
```

### Helper Files

* \{PROJECT_DIR\}/devbox.d/php81/php-fpm.conf
* \{PROJECT_DIR\}/devbox.d/php81/php.ini

You can modify these files to configure PHP or your PHP-FPM server

### Disabling the PHP Plugin

You can disable the PHP plugin by running `devbox add php --disable-plugin`, or by setting the `disable_plugin` field in your `devbox.json`:

```json
{
    "packages": {
        "php": {
            "version": "latest",
            "disable_plugin": true
        }
    }
}
```