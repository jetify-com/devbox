# devbox cache upload

Upload specified nix installable or nix packages in current project to cache.
If [installable] is provided, only that installable will be uploaded.
Otherwise, all packages in the project will be uploaded.
To upload to specific cache, use --to flag. Otherwise, a cache from
the cache provider will be used, if available.

```bash
devbox cache upload [installable] [flags]
```

## Aliases
upload, copy

## Options

| Option | Description |
| --- | --- |
| `-c, --config string` | path to directory containing a devbox.json config file |
| `-h, --help` | help for upload |
| `--to string` | URI of the cache to copy to |
| `-q, --quiet` | suppresses logs |