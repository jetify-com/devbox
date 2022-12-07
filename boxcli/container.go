// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

type containerCmdFlags struct {
	config configFlags
}

func ContainerCmd() *cobra.Command {

	command := &cobra.Command{
		Use:    "container",
		Hidden: true,
		Short:  "Contains interactions with OSI containers. Run 'devbox container -h' for a list of available commands.",
		Args:   cobra.MaximumNArgs(1),
	}

	command.AddCommand(exportCmd())

	return command
}

func exportCmd() *cobra.Command {
	flags := containerCmdFlags{}
	command := &cobra.Command{
		Use:   "export",
		Short: "Generate Dockerfile to build a container.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExportCmd(cmd, args, flags)
		},
	}
	flags.config.register(command)
	return command
}

func runExportCmd(_ *cobra.Command, args []string, flags containerCmdFlags) error {
	path, err := configPathFromUser(args, &flags.config)
	if err != nil {
		return err
	}
	// Check the directory exists.
	box, err := devbox.Open(path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}
	return box.ContainerExport(path)
}
