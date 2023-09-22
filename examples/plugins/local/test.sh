#!/bin/bash

if [ -z "$MY_FOO_VAR" ]; then
  echo "MY_FOO_VAR environment variable is not set."
  exit 1
else
  echo "MY_FOO_VAR is set to '$MY_FOO_VAR'"
fi

if [ -z "$MY_INIT_HOOK_VAR" ]; then
  echo "MY_INIT_HOOK_VAR environment variable is not set."
  exit 1
else
  echo "MY_INIT_HOOK_VAR is set to '$MY_INIT_HOOK_VAR'"
fi
