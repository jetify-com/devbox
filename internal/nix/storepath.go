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
	Output  string
}

// NewStorePathParts splits a Nix store path into its hash, name and version
// components in the same way that Nix does.
//
// See https://nixos.org/manual/nix/stable/language/builtins.html#builtins-parseDrvName
func NewStorePathParts(path string) StorePathParts {
	path = strings.TrimPrefix(path, "/nix/store/")
	// path is now <hash>-<name>-<version>[-output]

	hash, name := path[:32], path[33:]
	dashIndex := 0
	for i, r := range name {
		if dashIndex != 0 && !unicode.IsLetter(r) {
			version, output, _ := strings.Cut(name[i:], "-")
			return StorePathParts{Hash: hash, Name: name[:dashIndex], Version: version, Output: output}
		}
		dashIndex = 0
		if r == '-' {
			dashIndex = i
		}
	}
	return StorePathParts{Hash: hash, Name: name}
}
