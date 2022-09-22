---
title: PHP
---

### Detection

Devbox will automatically create a PHP Build plan whenever a `composer.json` or `composer.lock` file is detected in the project's root directory. 

Building a container with Devbox also requires a `public/index.php` file in your project directory. Running `devbox build` will return an error if one is not found. 

### Supported Versions

Devbox will attempt to detect the PHP version set in your `composer.json` file. The following major versions are supported (Devbox will always use the latest minor version for each major version):
* 7.4
* 8.0
* 8.1

If no version is set, Devbox will use 8.1 as the default version

### Included Nix Packages

* Depending on the detected PHP Version:
  * `php81`
  * `php80`
  * `php74`
* `phpPackages.composer`


### Default Stages

These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details

#### Install Stage

```bash
composer install --no-dev --no-ansi
```


#### Build Stage

*This stage is skipped for PHP projects*

#### Start Stage

```bash
php -S 0.0.0.0:8080 -t public
```
