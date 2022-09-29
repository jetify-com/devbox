# devbox build

Build an OCI image that can run as a container

## Synopsis

Builds your current source directory and devbox configuration as a Docker container. Devbox will create a plan for your container based on your source code, and then apply the packages and stage overrides in your devbox.json. 
 To learn more about how to configure your builds, see the [configuration reference](/docs/configuration)

```bash
devbox build [<dir>] [flags]
```

## Options

```text
      --engine string   Engine used to build the container: 'docker', 'podman' (default "docker")
  -h, --help            help for build
      --name string     name for the container (default "devbox")
      --no-cache        Do not use a cache
      --tags strings    tags for the container
```

## SEE ALSO

* [devbox](./devbox.md)	 - Instant, easy, predictable shells and containers

