---
title: Redis
---

Redis can be configured automatically using Devbox's built in Redis plugin. This plugin will activate automatically when you install Redis using `devbox add redis`

[**Example Repo**](https://github.com/jetpack-io/devbox-examples/tree/main/databases/redis)

## Adding Redis to your shell

`devbox add redis`, or in your Devbox.json

```json
    "packages": [
        "redis"
    ],
```

## Redis Plugin Details

The Redis plugin will automatically create the following configuration when you install Redis with `devbox add`

### Services

* redis

Use `devbox services start|stop [service]` to interact with services

### Helper Files

The following helper files will be created in your project directory:

* {PROJECT_DIR}/devbox.d/redis/redis.conf


### Environment Variables

```bash
REDIS_PORT=6379
REDIS_CONF=./devbox.d/redis/redis.conf
```

### Notes

Running `devbox services start redis` will start redis as a daemon in the background.

You can manually start Redis in the foreground by running `redis-server $REDIS_CONF --port $REDIS_PORT`.

Logs, pidfile, and data dumps are stored in `.devbox/virtenv/redis`. You can change this by modifying the `dir` directive in `devbox.d/redis/redis.conf`