// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

type addFlags struct {
	runtime bool
}

func AddCmd() *cobra.Command {
	flags := &addFlags{}

	command := &cobra.Command{
		Use:   "add <pkg>...",
		Short: "Add a new package to your devbox",
		Args:  cobra.MinimumNArgs(1),
		RunE:  addCmdFunc(flags),
	}

	command.Flags().BoolVarP(
		&flags.runtime, "runtime", "r", false, "The package is needed at runtime")
	return command
}

func addCmdFunc(flags *addFlags) runFunc {
	return func(cmd *cobra.Command, args []string) error {
		box, err := devbox.Open(".")
		if err != nil {
			return errors.WithStack(err)
		}

		if flags.runtime {
			return box.AddToRuntime(args...)
		} else {
			return box.Add(args...)
		}
	}
}
