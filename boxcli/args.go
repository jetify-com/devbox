// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"path/filepath"

	"github.com/pkg/errors"
)

// Functions that help parse arguments

// If args empty, defaults to the current directory
// Otherwise grabs the path from the first argument
func pathArg(args []string) string {
	if len(args) > 0 {
		p, err := filepath.Abs(args[0])
		if err != nil {
			panic(errors.WithStack(err)) // What even triggers this?
		}
		return p
	}
	return "."
}
