#!/bin/bash

expected="I AM SET (new value)"
custom_expected="I AM SET TO CUSTOM (new value)"
if [ "$MY_ENV_VAR" == "$expected" ] && [ "$MY_ENV_VAR_CUSTOM" == "$MY_ENV_VAR_CUSTOM" ]; then
  echo "Success! MY_ENV_VAR is set to '$MY_ENV_VAR'"
  echo "Success! MY_ENV_VAR_CUSTOM is set to '$MY_ENV_VAR_CUSTOM'"
else
  echo "MY_ENV_VAR environment variable is not set to '$expected'"
  echo "MY_ENV_VAR MY_ENV_VAR_CUSTOM variable is not set to '$MY_ENV_VAR_CUSTOM'"
  exit 1
fi
