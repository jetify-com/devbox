Inspired by https://github.com/amithgeorge/devbox-nodejs-repro-20230406

This example shows a wrapped binary calling setting an env variable (PATH) and
calling another wrapped binary without the PATH getting overwritten

## Steps

- devbox run run_test
- exit code should be 0
