# devbox secrets list

List all stored environment variables

## Synopsis

List all stored environment variables. If no environment flag is provided, variables in all environments will be listed.

```bash
devbox secrets list [flags]
```

## Options

```bash
      --environment string   Environment name, such as dev or prod (default "dev")
  -f, --format string        Display the key values in key=value format (default "table")
  -h, --help                 help for ls
      --org-id string        Organization id to namespace secrets by
      --project-id string    Project id to namespace secrets by
  -s, --show                 Display the value of each environment variable (secrets included)
```

## SEE ALSO

* [devbox_secrets](./devbox_secrets.md)  - Manage environment variables and secrets
