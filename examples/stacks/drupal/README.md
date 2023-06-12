# Drupal Stack

This example shows how to run a Drupal application in Devbox. It makes use of the PHP and Apache Plugins, while demonstrating how to configure a MariaDB instance to work with Devbox Cloud.

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/new?template=drupal)

## How to Run the example

To create the `devbox_drupal` database and example table, you should run:

`devbox run start`

To start all your services (PHP, MySQL, and NGINX) with an interactive terminal logs, run `devbox services up`. To stop the services, run `devbox services stop`
git 

## Configuration

Because the Nix Store is immutable, we need to store our configuration, data, and logs in a local project directory. This is stored in the `devbox.d` directory, in a subfolder for each of the packages that we will be installing.
