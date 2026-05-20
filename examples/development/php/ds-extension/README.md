# PHP with the `ds` extension

A minimal Devbox project that installs PHP 8.5 and the [`ds`](https://www.php.net/manual/en/book.ds.php)
extension and exercises `\Ds\Seq` from a short PHP script.

## Run it

```sh
devbox run run_test
```

The script verifies that `ds` is loaded with `extension_loaded('ds')` and then
constructs a `\Ds\Seq`, iterating its elements.
