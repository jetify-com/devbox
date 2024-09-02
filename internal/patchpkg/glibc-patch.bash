#!/bin/bash

set -euo pipefail

declare -r pkg # package that we're patching
declare -r out # nix output path that will contain the patched package

# Paths to this script's dependencies set by nix.
declare -r coreutils gnused ripgrep

# Explicitly declare the specific commands that this script depends on.
hash -p "$coreutils/bin/chmod" chmod
hash -p "$coreutils/bin/dirname" dirname
hash -p "$coreutils/bin/echo" echo
hash -p "$coreutils/bin/stat" stat
hash -p "$coreutils/bin/wc" wc
hash -p "$gnused/bin/sed" sed
hash -p "$ripgrep/bin/rg" rg

patch_store_path() {
	declare -r path="$1"
	declare -r perm=$(stat -c "%a" "$path")

	# sed creates a temporary sibling file for in-place edits, so we need to
	# ensure that the file's directory is writeable.
	declare -r dir="$(dirname "$path")"
	declare -r dperm=$(stat -c "%a" "$dir")

	echo "running sed file=$path file_perm=$perm dir=$dir dir_perm=$dperm"
	chmod u+w "$path" "$dir"
	sed -i -e "$sedexpr" "$path"
	chmod "$perm" "$path"
	chmod "$dperm" "$dir"
}

# -uu search ignored and hidden files
# -l list filenames
# -F exact substring search (faster, no escaping needed)
files="$(rg -uu -l -F "$pkg" "$out")"
count="$(echo "$files" | wc -l)"
sedexpr="s|$pkg|$out|g"
echo "patching files with old store path references count=$count sed=$sedexpr"
for f in $files; do
	patch_store_path "$f"
done
