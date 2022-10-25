
rustupHomeDir="$DEVBOX_CONFIG"/.rustup
mkdir -p $rustupHomeDir
export RUSTUP_HOME=$rustupHomeDir
export LIBRARY_PATH=$LIBRARY_PATH:"$DEVBOX_CONFIG/nix/profile/default/lib"

rustup default stable
