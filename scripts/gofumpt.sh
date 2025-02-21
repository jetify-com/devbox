#!/bin/bash

fd --extension go --exec-batch go tool gofumpt -extra -w

if [ -n "${CI:-}" ]; then
	git diff --exit-code
fi
