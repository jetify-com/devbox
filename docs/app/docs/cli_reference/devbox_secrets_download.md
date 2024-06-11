# devbox secrets download

Download environment variables into the specified file

## Synopsis

Download environment variables stored into the specified file (most commonly a .env file). The format of the file is one NAME=VALUE per line.

```bash
devbox secrets download <file1> [flags]
```

## Options

```bash
      --environment string   Environment name, such as dev or prod (default "dev")
  -f, --format string        File format: env or json (default "env")
  -h, --help                 help for download
      --org-id string        Organization id to namespace secrets by
      --project-id string    Project id to namespace secrets by
```

## SEE ALSO

* [devbox_secrets](./devbox_secrets.md)	 - Manage environment variables and secrets

