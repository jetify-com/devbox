// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

func SetupDirenv() *cobra.Command {
	command := &cobra.Command{
		Use:               "setup-direnv",
		Short:             "Prints out a shell script for setting up direnv integration.",
		Args:              cobra.MinimumNArgs(0),
		PersistentPreRunE: nixShellPersistentPreRunE,
		RunE:              setupDirenvFunc(),
	}

	return command
}

func setupDirenvFunc() runFunc {
	return func(cmd *cobra.Command, args []string) error {
		path := pathArg(args)
		box, err := devbox.Open(path, os.Stdout)
		if err != nil {
			return errors.WithStack(err)
		}

		return box.SetupDirenv()
	}
}
