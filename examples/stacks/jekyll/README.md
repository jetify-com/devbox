# Jekyll Example

[![Built with Devbox](https://jetify.com/img/devbox/shield_moon.svg)](https://jetify.com/devbox/docs/contributor-quickstart/)

[![Open In Devbox.sh](https://jetify.com/img/devbox/open-in-devbox.svg)](https://devbox.sh/open/templates/jekyll)

Inspired by [This Example](https://litchipi.github.io/nix/2023/01/12/build-jekyll-blog-with-nix.html)

## How to Use

1. Install [Devbox](https://www.jetify.com/devbox/docs/installing_devbox/)
1. Create a new project with:

    ```bash
    devbox create --template jekyll
    devbox install
    ```

1. Run `devbox shell` to install your packages and run the init hook
1. In the root directory, run `devbox run generate` to install and package the project with bundler
1. In the root directory, run `devbox run serve` to start the server. You can access the Jekyll example at `localhost:4000`

## Related Docs

* [Using Ruby with Devbox](https://www.jetify.com/devbox/docs/devbox_examples/languages/ruby/)
