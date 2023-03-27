# devbox services manager

Starts process manager with all supported services. This will start the services in your plugins, as well as the services in your `process-compose.yaml` file. 

For more information, see [Managing Services with Process Compose](../guides/services.md#managing-services)

```bash
devbox services manager [flags]
```

## Options

```bash
  -c, --config                  path to directory containing a devbox.json config file
  -h, --help                    help for manager
      --process-compose-file    path to process compose file or directory containing process compose-file.yaml|yml. Default is directory containing devbox.json

```

## SEE ALSO

* [devbox services](devbox_services.md)	 - Interact with devbox services

