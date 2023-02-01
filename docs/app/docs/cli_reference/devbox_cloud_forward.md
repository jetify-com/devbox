# devbox cloud forward
Port forwards a local port to a remote devbox cloud port. 

## Synopsis



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

Usage:
  devbox cloud forward \<local-port\>:\<remote-port\> | :\<remote-port\> | stop | list [flags]
  devbox cloud forward [command]

Available Commands:
  list        Lists all port forwards managed by devbox
  stop        Stops all port forwards managed by devbox

Flags:
  -h, --help   help for forward

Global Flags:
  -q, --quiet   suppresses logs.