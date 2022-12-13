// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/cloud"
)

type cloudShellCmdFlags struct {
	config configFlags
}

func CloudCmd() *cobra.Command {
	command := &cobra.Command{
		Use:    "cloud",
		Short:  "Remote development environments on the cloud",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(cloudShellCmd())
	return command
}

func cloudShellCmd() *cobra.Command {
	flags := cloudShellCmdFlags{}

	command := &cobra.Command{
		Use:   "shell",
		Short: "Shell into a cloud environment that matches your local devbox environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCloudShellCmd(&flags)
		},
	}

	flags.config.register(command)
	return command
}

func runCloudShellCmd(flags *cloudShellCmdFlags) error {
	box, err := devbox.Open(flags.config.path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}
	return cloud.Shell(box.ConfigDir())
}
