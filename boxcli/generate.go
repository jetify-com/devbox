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
	command.AddCommand(exportCmd())
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

func exportCmd() *cobra.Command {
	flags := &generateCmdFlags{}
	command := &cobra.Command{
		Use:   "devcontainer",
		Short: "Generate Dockerfile and devcontainer.json files",
		Long:  "Generate Dockerfile and devcontainer.json files necessary to run VSCode in remote container environments.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDevcontainerCmd(cmd, args, flags)
		},
	}
	flags.config.register(command)
	return command
}

func runDevcontainerCmd(_ *cobra.Command, args []string, flags *generateCmdFlags) error {
	path, err := configPathFromUser(args, &flags.config)
	if err != nil {
		return err
	}
	// Check the directory exists.
	box, err := devbox.Open(path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}
	return box.GenerateDevcontainer(path)
}
