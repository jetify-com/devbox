// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/boxcli/usererr"
)

// Functions that help parse arguments

// If args empty, defaults to the current directory
// Otherwise grabs the path from the first argument
func configPathFromUser(args []string, flags *configFlags) (string, error) {

	if flags.path != "" && len(args) > 0 {
		return "", usererr.New(
			"Cannot specify devbox.json's path via both --config and the command arguments. " +
				"Please use --config only.",
		)
	}

	if flags.path != "" {
		return flags.path, nil
	}

	if len(args) > 0 {
		return "", usererr.New(
			"devbox <command> <path> is deprecated, use devbox <command> --config <path> instead.",
		)
	}

	// current directory is ""
	return "", nil
}

func pathArg(args []string) string {
	if len(args) > 0 {
		p, err := filepath.Abs(args[0])
		if err != nil {
			panic(errors.WithStack(err)) // What even triggers this?
		}
		return p
	}
	return ""
}
