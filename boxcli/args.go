// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

// Functions that help parse arguments

// If args empty, defaults to the current directory
// Otherwise grabs the path from the first argument
func pathArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}
