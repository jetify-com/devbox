#!/usr/bin/env bash

projectDir=$(dirname $(readlink -f "${BASH_SOURCE:-$0}"))
echo "project dir is $projectDir"

rustupHomeDir="$projectDir"/.rustup
mkdir -p $rustupHomeDir
export RUSTUP_HOME=$rustupHomeDir
export LIBRARY_PATH=$LIBRARY_PATH:"$projectDir/nix/profile/default/lib"

rustup default 1.62.0
