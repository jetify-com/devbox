---
title: Ruby on Rails
description: An example of a Nix powered Ruby on Rails application with Devbox
---

This example demonstrates how to setup a simple Rails application. It makes use of the Ruby Plugin, and installs SQLite to use as a database.

[Example Repo](https://github.com/jetify-com/devbox/tree/main/examples/stacks/rails)

[![Open In Devspace](../../../static/img/open-in-devspace.svg)](https://auth.jetify.com/devspace/templates/rails)

## How To Run

Run `devbox shell` to install rails and prepare the project.

Once the shell starts, you can start the rails app by running:

```bash
cd blog
bin/rails server
```

## How to Recreate this Example

1. Create a new Devbox project with `devbox init`
2. Add the packages using

   ```bash
   devbox add ruby_3_1 bundler nodejs-19_x yarn sqlite
   ```

3. Start a devbox shell, and install the rails CLI with `gem install rails`
4. Create your Rails app by running the following in your Devbox Shell

   ```bash
   rails new blog
   ```
