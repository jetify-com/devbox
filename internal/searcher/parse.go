// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"strings"
)

// ParseVersionedPackage checks if the given package is a versioned package
// (`python@3.10`) and returns its name and version
func ParseVersionedPackage(pkg string) (string, string, bool) {
	lastIndex := strings.LastIndex(pkg, "@")
	if lastIndex == -1 {
		return "", "", false
	}
	return pkg[:lastIndex], pkg[lastIndex+1:], true
}
