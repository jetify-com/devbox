package nix

import (
	"strings"
	"unicode"
)

// StorePathParts are the constituent parts of
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
	for i, char := range name {
		if dashIndex != 0 && !unicode.IsLetter(char) {
			// Versions may contain dashes (2.39-52, 0-unstable-2024-01-01),
			// so only treat the last component as the output if it starts
			// with a letter.
			version, output := name[i:], ""
			if j := strings.LastIndex(version, "-"); j != -1 && j+1 < len(version) &&
				unicode.IsLetter(rune(version[j+1])) {
				version, output = version[:j], version[j+1:]
			}
			return StorePathParts{Hash: hash, Name: name[:dashIndex], Version: version, Output: output}
		}
		dashIndex = 0
		if char == '-' {
			dashIndex = i
		}
	}
	return StorePathParts{Hash: hash, Name: name}
}
