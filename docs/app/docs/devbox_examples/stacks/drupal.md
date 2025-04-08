---
title: Drupal
---

This example shows how to run a Drupal application in Devbox. It makes use of the PHP and Apache Plugins, while demonstrating how to configure a MariaDB instance to work with Devbox Cloud.

[Example Repo](https://github.com/jetify-com/devbox/tree/main/examples/stacks/drupal)


## How to Run

In this directory, run:

`devbox shell`

To start all your services (PHP, MySQL, and NGINX), run `devbox run start_services`. To stop the services, run `devbox run stop_services`

To create the `devbox_drupal` database and example table, you should run:

`mysql -u root < setup_db.sql`

To install Drupal and your dependencies, run `composer install`. The Drupal app will be installed in the `/web` directory, and you can configure your site by visiting `localhost/autoload` in your browser and following the interactive instructions

To exit the shell, use `exit`

## Installing the Umami Example

Run the `install-drupal.sh` script to install the Umami Drupal example. This is a good starter project for trying out and familiarizing yourself with Drupal

## Configuration

Because the Nix Store is immutable, we need to store our configuration, data, and logs in a local project directory. This is stored in the `devbox.d` directory, in a subfolder for each of the packages that we will be installing.
