#!/bin/bash

expected="I AM SET"
if [ "$MY_ENV_VAR" == "$expected" ]; then
  echo "Success! MY_ENV_VAR is set to '$MY_ENV_VAR'"
else
  echo "MY_ENV_VAR environment variable is not set to '$expected'"
  exit 1
fi
