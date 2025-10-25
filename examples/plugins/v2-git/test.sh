#!/bin/bash

expected="I AM SET (new value)"
custom_expected="I AM SET TO CUSTOM (new value)"
if [ "$MY_ENV_VAR" == "$expected" ] && [ "$MY_ENV_VAR_CUSTOM" == "$custom_expected" ]; then
  echo "Success! MY_ENV_VAR is set to '$MY_ENV_VAR'"
  echo "Success! MY_ENV_VAR_CUSTOM is set to '$MY_ENV_VAR_CUSTOM'"
else
  echo "ERROR: MY_ENV_VAR environment variable is not set to '$expected' OR MY_ENV_VAR_CUSTOM variable is not set to '$custom_expected'"
  exit 1
fi
