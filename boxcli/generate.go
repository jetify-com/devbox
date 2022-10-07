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
		Use:    "generate [<dir>]",
		Args:   cobra.MaximumNArgs(1),
		Hidden: true, // For debugging only
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateCmd(cmd, args, flags)
		},
	}

	registerConfigFlags(command, &flags.config)

	return command
}

func runGenerateCmd(cmd *cobra.Command, args []string, flags *generateCmdFlags) error {
	path := pathArg(args, &flags.config)

	// Check the directory exists.
	box, err := devbox.Open(path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Generate()
}
