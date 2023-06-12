## nginx-1.22.1

### nginx NOTES:
nginx is configured to use devbox.d/nginx/nginx.conf

To customize:
* Use $NGINX_CONFDIR to change the configuration directory
* Use $NGINX_LOGDIR to change the log directory
* Use $NGINX_PIDDIR to change the pid directory
* Use $NGINX_RUNDIR to change the run directory
* Use $NGINX_SITESDIR to change the sites directory
* Use $NGINX_TMPDIR to change the tmp directory. Use $NGINX_USER to change the user
* Use $NGINX_GROUP to customize.

### Services:
* nginx

Use `devbox services start|stop [service]` to interact with services

### This configuration creates the following helper files:
* devbox.d/nginx/nginx.conf
* devbox.d/nginx/fastcgi.conf

### This configuration sets the following environment variables:
* NGINX_CONFDIR=<root_dir>/devbox.d/nginx

To show this information, run `devbox info nginx`

