---
title: Redis with Upstash
sidebar_position: 6
---

[Upstash](https://upstash.com) provides a managed Key-value store that is compatible with the Redis API. You can use Upstash Redis as a cache or key-value store for serverless or stateless deployments like Jetify Cloud. For more information, consult the [Upstash Docs](https://upstash.com/docs/introduction)

## Using Upstash Redis with Jetify Cloud

* [Create a Redis Database](https://upstash.com/docs/redis/overall/getstarted) in Upstash. Select the Database's name and region, and then click the Create button
  * If you don't already have an Upstash account, you can create a free one to test with Jetify Cloud
* After clicking the Create button, you'll see a page with the connection details for your Database. Copy the Endpoint, Password, and Port -- you'll need these to connect from Jetify Cloud. 

![Upstash Dashboard after clicking Create](../../../../static/img/upstash.png)

* Go to the Jetify Dashboard for your project, and navigate to Secrets. Create the following Secrets in the `Prod` environment: 
  * `REDIS_HOST`: your Upstash DB Endpoint URL
  * `REDIS_PASSWORD`: your Upstash DB Password
  * `REDIS_PORT`: your Upstash DB Port 

:::info
 If you want to use your Database locally or in a preview environment, you can also set these environment variables for the `dev` and `preview` environment
:::

![Secrets set in the Jetify Cloud](../../../../static/img/upstash-secrets.png)

When you deploy your application, Devbox will automatically set these secrets as env variables in your environment. You can then access them using any redis client. 

For example, if you are connecting from a Python app, you can do something like the following to connect from a Redis client.

```python
import os
import redis

redis_cache = redis.StrictRedis(
    host=os.getenv('REDIS_HOST'),
    password=os.getenv('REDIS_PASSWORD'),
    port=os.getenv('REDIS_PORT'),
    ssl=True
)
```