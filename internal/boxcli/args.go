// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"path/filepath"

	"github.com/pkg/errors"
)

// Functions that help parse arguments

func pathArg(args []string) string {
	if len(args) > 0 {
		p, err := filepath.Abs(args[0])
		if err != nil {
			// Can occur when the current working directory cannot be determined.
			panic(errors.WithStack(err))
		}
		return p
	}
	return ""
}
