// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devpkg

import (
	"strings"
)

// ParseVersionedPackage checks if the given package is a versioned package (`python@3.10`)
// and returns its name and version
func ParseVersionedPackage(pkg string) (string, string, bool) {
	name, version, found := strings.Cut(pkg, "@")
	return name, version, found && name != "" && version != ""
}
