// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cloud"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/envir"
)

type cloudShellCmdFlags struct {
	config configFlags

	githubUsername string
}

func cloudCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "cloud",
		Short: "[Preview] Remote development environments on the cloud",
		Long: "Remote development environments on the cloud. All cloud commands " +
			"are currently in developer preview and may have some rough edges. " +
			"Please report any issues to https://github.com/jetpack-io/devbox/issues",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(cloudShellCmd())
	command.AddCommand(cloudInitCmd())
	command.AddCommand(cloudPortForwardCmd())
	return command
}

func cloudInitCmd() *cobra.Command {
	flags := cloudShellCmdFlags{}
	command := &cobra.Command{
		Use:    "init",
		Hidden: true,
		Short:  "Create a Cloud VM without connecting to its shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCloudInit(cmd, &flags)
		},
	}
	flags.config.register(command)
	return command
}

func cloudShellCmd() *cobra.Command {
	flags := cloudShellCmdFlags{}

	command := &cobra.Command{
		Use:   "shell",
		Short: "[Preview] Shell into a cloud environment that matches your local devbox environment",
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
		Use:   "forward <local-port>:<remote-port> | :<remote-port> | stop | list",
		Short: "[Preview] Port forward a local port to a remote devbox cloud port",
		Long: "Port forward a local port to a remote devbox cloud port. If 0 or " +
			"no local port is specified, we find a suitable local port. Use 'stop' " +
			"to stop all port forwards.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ports := strings.Split(args[0], ":")

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
	command.AddCommand(cloudPortForwardStopCmd())
	return command
}

func cloudPortForwardStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop all port forwards managed by devbox",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cloud.PortForwardTerminateAll()
		},
	}
}

func cloudPortForwardList() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all port forwards managed by devbox",
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
	// calling `devbox cloud shell` when already in the VM is not allowed.
	if envir.IsDevboxCloud() {
		return shellInceptionErrorMsg("devbox cloud shell")
	}

	box, err := devbox.Open(&devopt.Opts{
		Dir:         flags.config.path,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	return cloud.Shell(cmd.Context(), cmd.ErrOrStderr(), box.ProjectDir(), flags.githubUsername)
}

func runCloudInit(cmd *cobra.Command, flags *cloudShellCmdFlags) error {
	// calling `devbox cloud init` when already in the VM is not allowed.
	if envir.IsDevboxCloud() {
		return shellInceptionErrorMsg("devbox cloud init")
	}

	box, err := devbox.Open(&devopt.Opts{
		Dir:         flags.config.path,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	_, vmhostname, _, err := cloud.InitVM(cmd.Context(), cmd.ErrOrStderr(), box.ProjectDir(), flags.githubUsername)
	if err != nil {
		return err
	}
	// printing vmHostname so that the output of devbox cloud init can be read by
	// devbox extension
	fmt.Fprintln(cmd.ErrOrStderr(), vmhostname)
	return nil
}
