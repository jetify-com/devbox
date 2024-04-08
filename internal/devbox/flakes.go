// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"strings"
)

// getLocalFlakesDirs searches packages and returns list of directories
// of local flakes that are mentioned in config.
// e.g., path:./my-flake#packageName -> ./my-flakes
func (d *Devbox) getLocalFlakesDirs() []string {
	localFlakeDirs := []string{}

	// searching through installed packages to get location of local flakes
	for _, pkg := range d.PackageNames() {
		// filtering local flakes packages
		if strings.HasPrefix(pkg, "path:") {
			pkgDirAndName, _ := strings.CutPrefix(pkg, "path:")
			pkgDir := strings.Split(pkgDirAndName, "#")[0]
			localFlakeDirs = append(localFlakeDirs, pkgDir)
		}
	}
	return localFlakeDirs
}
