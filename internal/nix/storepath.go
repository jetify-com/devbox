package nix

import (
	"strings"
	"unicode"
)

// storePath are the constituent parts of
// /nix/store/<hash>-<name>-<version>
//
// This is a helper struct for analyzing the string representation
type StorePathParts struct {
	Hash    string
	Name    string
	Version string
}

// NewStorePathParts splits a Nix store path into its hash, name and version
// components in the same way that Nix does.
//
// See https://nixos.org/manual/nix/stable/language/builtins.html#builtins-parseDrvName
//
// TODO: store paths can also have `-{output}` suffixes, which need to be handled below.
func NewStorePathParts(path string) StorePathParts {
	path = strings.TrimPrefix(path, "/nix/store/")
	// path is now <hash>-<name>-<version

	hash, name := path[:32], path[33:]
	dashIndex := 0
	for i, r := range name {
		if dashIndex != 0 && !unicode.IsLetter(r) {
			return StorePathParts{Hash: hash, Name: name[:dashIndex], Version: name[i:]}
		}
		dashIndex = 0
		if r == '-' {
			dashIndex = i
		}
	}
	return StorePathParts{Hash: hash, Name: name}
}
