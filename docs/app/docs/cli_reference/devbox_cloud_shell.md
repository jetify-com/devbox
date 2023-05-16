# devbox cloud shell
[Preview] Shell into a remote environment on Devbox Cloud.


## Synopsis
When run in a directory with a `devbox.json` file, this command will start a VM, sync your local files, and create an environment using the packages and configuration in your `devbox.json`.

If a `devbox.json` file is not detected in your current directory, the Devbox CLI will attempt to find a `devbox.json` file in parent directories, and return an error if one is not located.

To authenticate with Devbox Cloud, you must have a Github Account with a linked SSH key. Devbox will attempt to automatically detect your Github username and public key, and prompt you if one cannot be identified,

For more details on how to use Devbox Cloud, consult the [Getting Started Guide](../devbox_cloud/getting_started.mdx)


```bash
# Start a Cloud Shell
devbox cloud shell
```

## Options
<!-- Markdown Table of Options  -->
| Option | Description |
| --- | --- |
| `-c, --config string` | path to directory containing a devbox.json config file |
| `-h, --help` | help for shell |
| `-u, --username string` | Github username to use for ssh |
| `-q, --quiet` | suppresses logs. |
