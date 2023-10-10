#!/bin/bash

find . -name '*.go' -exec gofumpt -extra -w {} \+

if [ -n "${CI:-}" ]; then
  git diff --exit-code
fi
