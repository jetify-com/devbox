
# TODO this only works when devbox shell is started in this directory. Using
# the --config flag to start the shell will break this.
# We could inject $JETPACK_CONFIG env-var into the shell environment to replace this.
projectDir=$(dirname $(readlink -f "$0"))
echo "project dir is $projectDir"

rustupHomeDir="$projectDir"/.rustup
mkdir -p $rustupHomeDir
export RUSTUP_HOME=$rustupHomeDir
export LIBRARY_PATH=$LIBRARY_PATH:"$projectDir/nix/profile/default/lib"

rustup default stable
