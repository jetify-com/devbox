---
title: Redis
---

The easiest way to configure Redis with Devbox is by adding a custom `redis.conf` to your project, and then starting `redis-server` by pointing it to the conf file. 

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/databases/redis)

## Adding Redis to your shell

`devbox add redis`, or in your Devbox.json

```json
    "packages": [
        "redis"
    ],
```

## Example redis.conf

```conf
##### NETWORK #####
bind 127.0.0.1 -::1
protected-mode yes
port 6379

##### TLS/SSL  #####   

# Use default settings

##### GENERAL ##### 

daemonize yes
pidfile conf/redis/redis.pid
logfile redis.log

##### SNAPSHOTTING #####

dir conf/redis/data/
```

If this config is stored under your project folder at `conf/redis`, then you can start redis with the correct settings by running:
`redis-server conf/redis` from within your devbox shell
