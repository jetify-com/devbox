# devbox secrets set

Securely store one or more environment variables

## Synopsis

Securely store one or more environment variables.

```bash
devbox secrets set <NAME1>=<value1> [<NAME2>=<value2>]... [flags]
```

## Options

```bash
      --environment string   Environment name, such as dev or prod (default "dev")
  -h, --help                 help for set
      --org-id string        Organization id to namespace secrets by
      --project-id string    Project id to namespace secrets by
```

## SEE ALSO

* [devbox_secrets](./devbox_secrets.md)  - Manage environment variables and secrets
