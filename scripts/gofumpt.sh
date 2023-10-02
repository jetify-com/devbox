#!/bin/bash

mkdir -p dist/tools
export GOBIN="$PWD/dist/tools"
go install mvdan.cc/gofumpt@latest

find . -name '*.go' -exec "$GOBIN/gofumpt" -extra -w {} \+

if [ -n "${CI:-}" ]; then
  git diff --exit-code
fi
