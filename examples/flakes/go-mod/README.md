# Building a Go Module with Flakes

This flake shows how to build a custom Go module and add it to your Devbox project. In this case, we're building the [Ory CLI](https://github.com/ory/cli)

This example uses `buildGoModule` from Nix to build the module as a package in our Flake. You can view the flake.nix file in the ory-cli folder to see a commented example of how this function is used.

We import the ory CLI in our project by adding it to our packages in `devbox.json`:

```json
{
  "packages": [
    "path:ory-cli"
  ],
   ...
}
```

Note: you will need [Devbox 0.4.7](https://www.jetify.com/blog/devbox-0-4-7/) or later for this to work. You can use this as an example to create your own templates.

For more details on using Flakes with Devbox, read our post on [Using Nix Flakes with Devbox](https://www.jetify.com/blog/using-nix-flakes-with-devbox/)
