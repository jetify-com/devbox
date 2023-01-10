// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
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
	command.AddCommand(cloudPortForwardCmd())
	return command
}

func cloudShellCmd() *cobra.Command {
	flags := cloudShellCmdFlags{}

	command := &cobra.Command{
		Use:   "shell",
		Short: "Shell into a cloud environment that matches your local devbox environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCloudShellCmd(cmd, &flags)
		},
	}

	flags.config.register(command)
	return command
}

func cloudPortForwardCmd() *cobra.Command {
	command := &cobra.Command{
		Use:    "port-forward <local-port>:<remote-port>",
		Short:  "Port forwards a local port to a remote devbox cloud port",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ports := strings.Split(args[0], ":")
			if len(ports) != 2 {
				return usererr.New("Invalid port format. Expected <local-port>:<remote-port>")
			}
			err := cloud.PortForward(ports[0], ports[1])
			if err != nil {
				return errors.WithStack(err)
			}
			cmd.PrintErrf("Port forwarding %s:%s\n", ports[0], ports[1])
			return nil
		},
	}

	return command
}

func runCloudShellCmd(cmd *cobra.Command, flags *cloudShellCmdFlags) error {
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}
	return cloud.Shell(box.ProjectDir(), cmd.ErrOrStderr())
}
