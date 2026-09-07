{ stdenv, lib, bash, writeShellScriptBin }:

writeShellScriptBin "hello" ''
  echo "Hello from a custom Nix package!"
''
