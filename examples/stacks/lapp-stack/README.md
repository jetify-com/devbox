# LAPP Stack

This example shows how to build a simple application using Apache, PHP, and PostgreSQL. It uses Devbox Plugins for all 3 packages to simplify configuration.

[![Open In Devspace](https://www.jetify.com/img/devbox/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/lapp-stack)

## How to Run

The following steps may be done inside or outside a devbox shell.

1. Initialize a database by running `devbox run init_db`.
1. Create the database and load the test data by using `devbox run create_db`.
1. Start Apache, PHP-FPM, and Postgres in the background by run `devbox services start`.
1. You can now test the app using `localhost:8080` to hit the Apache Server. If you want Apache to listen on a different port, you can change the `HTTPD_PORT` environment variable in the Devbox init_hook.

### How to Recreate this Example

1. Create a new project with:
    ```bash
    devbox create --template lapp-stack
    devbox install
    ```

1. Update `devbox.d/apache/httpd.conf` to point to the directory with your PHP files. You'll need to update the `DocumentRoot` and `Directory` directives.
1. Follow the instructions above in the How to Run section to initialize your project.

### Related Docs

* [Using PHP with Devbox](https://www.jetify.com/devbox/docs/devbox_examples/languages/php/)
* [Using Apache with Devbox](https://www.jetify.com/devbox/docs/devbox_examples/servers/apache/)
* [Using PostgreSQL with Devbox](https://www.jetify.com/devbox/docs/devbox_examples/databases/postgres/)
