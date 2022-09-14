# Nginx Static Planner

**Warning:** this planner is experimental. The detection, API, and nix derivations may all change.

## How detection works

This planner looks for `nginx.conf` or `shell-nginx.conf` in your devbox.json
directory. It defaults to `nginx.conf` for building and `shell-nginx.conf` for 
shell. If it can't find the correct config it uses the other one.

## How nginx works in shell

To run shell nginx you can use the `shell-nginx` wrapper. This wrapper calls nginx 
with a few options. If you want to see what this wrapper does, use `cat $(which shell-nginx)`

In shell everything is local so you should avoid pointing to assets or files outside 
the directory because the nix shell might not have access. For example your root
maybe be described as `root ./static/;` and access log `access_log /tmp/access.log;`

Shell nginx is compiled to use `/tmp/cache` as the default cache directory. See 
[nginx planner](nginx_planner.go) for implementation.

## How nginx works when building

nginx is installed as a package on the runtime image. File contents are copied over
to the /app working directory. A typical root directive would be `root /app/static;`
The container is configured to have a user and group named nginx which you can use
with the `user nginx nginx;` directive.
