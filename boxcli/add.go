// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

type addCmdFlags struct {
	config configFlags
}

func AddCmd() *cobra.Command {
	flags := addCmdFlags{}

	command := &cobra.Command{
		Use:               "add <pkg>...",
		Short:             "Add a new package to your devbox",
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: nixShellPersistentPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			return addCmdFunc(cmd, args, flags)
		},
	}

	flags.config.register(command)
	return command
}

func addCmdFunc(_ *cobra.Command, args []string, flags addCmdFlags) error {
	box, err := devbox.Open(flags.config.path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Add(args...)
}
