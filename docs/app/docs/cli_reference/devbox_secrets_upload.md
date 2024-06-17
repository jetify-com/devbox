# devbox secrets upload

Upload variables defined in a .env file

## Synopsis

Upload variables defined in one or more .env files. The files should have one NAME=VALUE per line.

```bash
devbox secrets upload <file1> [<fileN>]... [flags]
```

## Options

```bash
      --environment string   Environment name, such as dev or prod (default "dev")
  -f, --format string        File format: env or json (default "env")
  -h, --help                 help for upload
      --org-id string        Organization id to namespace secrets by
      --project-id string    Project id to namespace secrets by
```

## SEE ALSO

* [devbox_secrets](./devbox_secrets.md)  - Manage environment variables and secrets
