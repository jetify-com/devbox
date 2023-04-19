# Adding a custom PHP Extension

This example shows how to add a custom PHP extension to a PHP package using Flakes. This uses an extension built from the [PHP Skeleton Extension](https://github.com/improved-php-library/skeleton-php-ext) from [Improved PHP Library](https://github.com/improved-php-library)

To test this example:

1. Run `devbox shell` to start your shell
2. Start PHP in interactive mode with `php -a`
3. Run `echo skeleton_nop("Hello World");` to test the extension. This should print `Hello World` to the screen
