// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

type generateCmdFlags struct {
	config configFlags
}

func GenerateCmd() *cobra.Command {
	flags := &generateCmdFlags{}

	command := &cobra.Command{
		Use:    "generate",
		Args:   cobra.MaximumNArgs(1),
		Hidden: true, // For debugging only
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateCmd(cmd, args, flags)
		},
	}

	flags.config.register(command)

	return command
}

func runGenerateCmd(_ *cobra.Command, args []string, flags *generateCmdFlags) error {
	path, err := configPathFromUser(args, &flags.config)
	if err != nil {
		return err
	}

	// Check the directory exists.
	box, err := devbox.Open(path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Generate()
}
