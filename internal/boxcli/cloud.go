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

	githubUsername string
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
	command.Flags().StringVarP(
		&flags.githubUsername, "username", "u", "", "Github username to use for ssh",
	)
	return command
}

func cloudPortForwardCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "port-forward <local-port>:<remote-port> | <port> | :<remote-port> | terminate",
		Short: "Port forwards a local port to a remote devbox cloud port",
		Long: "Port forwards a local port to a remote devbox cloud port. If a " +
			"single port is specified, it is used for local and remote. If no local" +
			" port is specified, we find a suitable local port. Use 'terminate' to " +
			"terminate all port forwards.",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ports := []string{}
			if strings.ContainsRune(args[0], ':') {
				ports = strings.Split(args[0], ":")
			} else {
				ports = append(ports, args[0], args[0])
			}

			if len(ports) != 2 {
				return usererr.New("Invalid port format. Expected <local-port>:<remote-port>")
			}
			localPort, err := cloud.PortForward(ports[0], ports[1])
			if err != nil {
				return errors.WithStack(err)
			}
			cmd.PrintErrf(
				"Port forwarding %s:%s\nTo view in browser, visit http://localhost:%[1]s\n",
				localPort,
				ports[1],
			)
			return nil
		},
	}
	command.AddCommand(cloudPortForwardList())
	command.AddCommand(cloudPortForwardTerminateAllCmd())
	return command
}

func cloudPortForwardTerminateAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "terminate",
		Short:  "Terminates all port forwards managed by devbox",
		Hidden: true,
		Args:   cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cloud.PortForwardTerminateAll()
		},
	}
}

func cloudPortForwardList() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists all port forwards managed by devbox",
		Hidden:  true,
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := cloud.PortForwardList()
			if err != nil {
				return errors.WithStack(err)
			}
			for _, p := range l {
				cmd.Println(p)
			}
			return nil
		},
	}
}

func runCloudShellCmd(cmd *cobra.Command, flags *cloudShellCmdFlags) error {
	if devbox.IsDevboxShellEnabled() {
		return shellInceptionErrorMsg("devbox cloud shell")
	}

	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}
	return cloud.Shell(cmd.ErrOrStderr(), box.ProjectDir(), flags.githubUsername)
}
