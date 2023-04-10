# devbox cloud forward

Port forwards a local port to a remote devbox cloud port. If 0 or no local port is specified, we find a suitable local port. Use 'stop' to stop all port forwards.

```bash
devbox cloud forward <local-port>:<remote-port> | :<remote-port> | stop | list [flags]
```

## Examples

```bash
# Connect remote port 3000 to an automatically assigned local port
$ devbox cloud forward :3000
Port forwarding 54653:3000
To view in browser, visit http://localhost:54653
```

```bash
# Connect remote port 3000 to local port 3000
$ devbox cloud forward 3000:3000
Port forwarding 3000:3000
To view in browser, visit http://localhost:3000
```

```bash
# Close all open port-forwards
$ devbox cloud forward stop
```

## Options

<!-- Markdown Table of Options -->
| Option | Description |
| --- | --- |
| `-h, --help` | help for forward |
| `-q, --quiet` | suppresses logs |

## SEE ALSO

* [devbox cloud](devbox_cloud.md)	 - [Preview] Remote development environments on the cloud
* [devbox cloud forward list](devbox_cloud_forward_list.md)	 - Lists all port forwards managed by devbox
* [devbox cloud forward stop](devbox_cloud_forward_stop.md)	 - Stops all port forwards managed by devbox

