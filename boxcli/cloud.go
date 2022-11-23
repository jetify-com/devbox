// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/cloud"
)

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
	command := &cobra.Command{
		Use:   "shell",
		Short: "Shell into a cloud environment that matches your local devbox environment",
		RunE:  runCloudShellCmd,
	}

	return command
}

func runCloudShellCmd(cmd *cobra.Command, args []string) error {
	return cloud.Shell()
}
