# Django 

This example demonstrates how to configure and run a Django app using Devbox. It installs Python, PostgreSQL, and uses `pip` to install your Python dependencies in a virtual environment.

[Example Repo](https://github.com/jetpack-io/devbox-examples/tree/main/stacks/django)

[![Open In Devbox.sh](https://jetpack.io/img/devbox/open-in-devbox.svg)](https://devbox.sh/github.com/jetpack-io/devbox-examples?folder=stacks/django)

## How to Use

1. Install [Devbox](https://www.jetpack.io/devbox/docs/installing_devbox/)
1. Run `devbox shell` to install your packages and run the init_hook. This will activate your virtual environment and install Django.
1. Initialize PostgreSQL with `devbox run initdb`.
1. In the root directory, run `devbox run create_db` to create the database and run your Django migrations.
1. In the root directory, run `devbox run server` to start the server. You can access the Django example at `localhost:8000`.

## How to Create this Example from Scratch

1. Install [Devbox](https://www.jetpack.io/devbox/docs/installing_devbox/).
1. Run `devbox init` to create a new Devbox project in your directory.
1. Install Python and PostgreSQL with `devbox install python python310Packages.pip openssl postgresql`. This will also install the Devbox plugins for pip (which sets up your .venv directory) and PostgreSQL.
1. Copy the requirements.txt and `todo_project` directories.
1. Initialize your Postgres database with `devbox run -- initdb`.
1. Start a devbox shell with `devbox shell`, then activate your virtual environment and install your requirements using the command below.

    ```bash
    source $VENV_DIR/bin/activate
    pip install -r requirements.txt
    ```

    You can also add these lines to your `init_hook` so they run whenever you start a devbox shell.

1. Run the following script to setup your database. This script will start the Postgres service, create the DB and user for your project, and run the Django project's migrations.

   ```bash
    echo "Creating DB"
    devbox services restart postgresql
    dropdb --if-exists todo_db
    createdb todo_db
    psql todo_db -c "CREATE USER todo_user WITH PASSWORD 'secretpassword';"
    python todo_project/manage.py makemigrations
    python todo_project/manage.py migrate
   ```

   You can add this as a devbox script in your `devbox.json` file, so you can replicate the setup on other machines.

1. You can now start your Django server by running the following command.

   ```bash
   python todo_project/manage.py runserver
   ```