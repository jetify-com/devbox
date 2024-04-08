# Rails Example in Devbox

This example demonstrates how to setup a simple Rails application. It makes use of the Ruby Plugin, and installs SQLite to use as a database.

[![Open In Devbox.sh](https://jetify.com/img/devbox/open-in-devbox.svg)](https://devbox.sh/open/templates/rails)

## How To Run

Run `devbox shell` to install rails and prepare the project.

Once the shell starts, you can start the rails app by running:

```bash
cd blog
bin/rails server
```

## How to Recreate this Example

1. Create a new Devbox project with `devbox create --template rails`
2. Add the packages using

   ```bash
   devbox install
   ```

3. Run `devbox shell`, which will install the rails CLI with `gem install rails`
4. Create your Rails app by running the following in your Devbox Shell

   ```bash
   rails new blog
   ```

## Related Docs

* [Using Ruby with Devbox](https://www.jetify.com/devbox/docs/devbox_examples/languages/ruby/)
