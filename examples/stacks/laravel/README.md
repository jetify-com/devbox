# Laravel

Laravel is a powerful web application framework built with PHP. It's a great choice for building web applications and APIs.

This example shows how to build a simple Laravel application backed by MariaDB and Redis. It uses Devbox Plugins for all 3 Nix packages to simplify configuration

[![Open In Devspace](https://www.jetify.com/img/devbox/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/laravel)

## How to Run

1. Install [Devbox](https://www.jetify.com/devbox/docs/installing_devbox/)

1. Create a new Laravel App by running `devbox create --template laravel`. This will create a new Laravel project in your current directory.

1. Start your MariaDB and Redis services by running `devbox services up`.
   1. This step will also create an empty MariaDB Data Directory and initialize your database with the default settings
   2. This will also start the php-fpm service for serving your PHP project over fcgi. Learn more about [PHP-FPM](https://www.php.net/manual/en/install.fpm.php)

1. Create the laravel database by running `devbox run db:create`, and then run Laravel's initial migrations using `devbox run db:migrate`

1. You can now start the artisan server by running `devbox run serve:dev`. This will start the server on port 8000, which you can access at `localhost:8000`

1. If you're using Laravel on Devbox Cloud, you can test the app by appending `/port/8000` to your Devbox Cloud URL

1. For more details on building and developing your Laravel project, visit the [Laravel Docs](https://laravel.com/docs/10.x)


## How to Recreate this Example

### Creating the Laravel Project

1. Create a new project with `devbox init`

2. Add the packages using the command below. Installing the packages with `devbox add` will ensure that the plugins are activated:

    ```bash
    devbox add mariadb@latest, php@8.1, nodejs@18, redis@latest, php81Packages.composer@latest
    ```

3. Run `devbox shell` to start your shell. This will also initialize your database by running `initdb` in the init hook.

4. Create your laravel project by running:

    ```bash
    composer create-project laravel/laravel tmp

    mv tmp/* tmp/.* .
    ```

### Setting up MariaDB

To use MariaDB, you need to create the default Laravel database. You can do this by running the following commands in your `devbox shell`:

```bash
# Start the MariaDB service
devbox services up mariadb -b

# Create the database
mysql -u root -e "CREATE DATABASE laravel;"

# Once you're done, stop the MariaDB service
devbox services stop mariadb
```
