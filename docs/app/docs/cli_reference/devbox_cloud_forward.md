## devbox cloud forward

[Preview] Port forwards a local port to a remote devbox cloud port

### Synopsis

Port forwards a local port to a remote devbox cloud port. If 0 or no local port is specified, we find a suitable local port. Use 'stop' to stop all port forwards.

```
devbox cloud forward <local-port>:<remote-port> | :<remote-port> | stop | list [flags]
```

### Options

```
  -h, --help   help for forward
```

### Options inherited from parent commands

```
  -q, --quiet   suppresses logs
```

### SEE ALSO

* [devbox cloud](devbox_cloud.md)	 - [Preview] Remote development environments on the cloud
* [devbox cloud forward list](devbox_cloud_forward_list.md)	 - Lists all port forwards managed by devbox
* [devbox cloud forward stop](devbox_cloud_forward_stop.md)	 - Stops all port forwards managed by devbox

