# Temporal

[![Built with Devbox](https://www.jetify.com/devbox/img/shield_galaxy.svg)](https://www.jetify.com/devbox/docs/contributor-quickstart/)

Example devbox for testing and developing Temporal workflows using Temporalite and the Python Temporal SDK.

For more details, check out:

* [Temporal.io](https://temporal.io/)
* [Temporalite](https://github.com/temporalio/temporalite)
* [Temporal Python SDK](https://github.com/temporalio/sdk-python)
* [Temporal Python Samples](https://github.com/temporalio/sample-python)

## Starting Temporal

```bash
devbox run start-temporal
```

This will start the temporalite server for testing.

* You can view the WebUI at `localhost:8233`
* By default, Temporal will listen for activities/requests on port `7233`

## Starting a Devbox Shell

```bash
devbox shell
```

This will activate a virtual environment and install the Temporal Python SDK.

## Testing the Temporal Workflows

From inside your `devbox shell`

```bash
cd temporal_example/hello
python run hello_activity.py
```

This should start the workflow using temporalite.
