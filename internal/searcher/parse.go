// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"strings"
)

// ParseVersionedPackage checks if the given package is a versioned package
// (`python@3.10`) and returns its name and version
func ParseVersionedPackage(versionedName string) (name, version string, found bool) {
	// use the last @ symbol as the version delimiter, some packages have @ in the name
	atSymbolIndex := strings.LastIndex(versionedName, "@")
	if atSymbolIndex == -1 {
		return "", "", false
	}
	if atSymbolIndex == len(versionedName)-1 {
		// This case handles packages that end with `@` in the name
		// example: `emacsPackages.@`
		return "", "", false
	}

	// Common case: package@version
	name, version = versionedName[:atSymbolIndex], versionedName[atSymbolIndex+1:]
	return name, version, true
}
