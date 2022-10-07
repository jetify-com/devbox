// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

// Functions that help parse arguments

// If args empty, defaults to the current directory
// Otherwise grabs the path from the first argument
func pathArg(args []string, flags *configFlags) string {

	if flags.path != currentDir {
		if len(args) > 0 {
			// Choose the --config flag because config argument is being deprecated
			fmt.Printf(
				"%s You are specifying the config path as an argument and using the --config flag. "+
					"Choosing to ignore the argument and use the flag.\n",
				color.HiYellowString("Warning:"),
			)
		}
		return flags.path
	}

	if len(args) > 0 {
		fmt.Printf(
			"%s please use the --config or -c flag to specify the path to the devbox.json config. "+
				"We are deprecating the previous way of specifying this path as an argument to the command.\n",
			color.HiYellowString("Warning:"),
		)
		p, err := filepath.Abs(args[0])
		if err != nil {
			panic(errors.WithStack(err)) // What even triggers this?
		}
		return p
	}
	return "."
}
