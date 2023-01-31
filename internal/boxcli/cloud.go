// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

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
		Use:   "cloud",
		Short: "[Preview] Remote development environments on the cloud",
		Long: "Remote development environments on the cloud. All cloud commands " +
			"are currently in developer preview and may have some rough edges. " +
			"Please report any issues to https://github.com/jetpack-io/devbox/issues",
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
		Short: "[Preview] Port forwards a local port to a remote devbox cloud port",
		Long: "Port forwards a local port to a remote devbox cloud port. If 0 or " +
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
	command.AddCommand(cloudPortForwardAuto())
	command.AddCommand(cloudPortForwardStopCmd())
	return command
}

func cloudPortForwardStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stops all port forwards managed by devbox",
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
		Short:   "Lists all port forwards managed by devbox",
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

func cloudPortForwardAuto() *cobra.Command {
	return &cobra.Command{
		Use:    "auto",
		Short:  "Automatically port forwards all ports managed by devbox",
		Hidden: true,
		Args:   cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			box, err := devbox.Open("", cmd.ErrOrStderr())
			if err != nil {
				return errors.WithStack(err)
			}
			if err = cloud.AutoPortForward(cmd.Context(), cmd.ErrOrStderr(), box.ProjectDir()); err != nil {
				return err
			}
			done := make(chan os.Signal, 1)
			signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
			fmt.Println("Listening, press ctrl+c to end...")
			<-done // Will block here until user hits ctrl+c
			return nil
		},
	}
}

func runCloudShellCmd(cmd *cobra.Command, flags *cloudShellCmdFlags) error {
	// calling `devbox cloud shell` when already in the VM is not allowed.
	if region := os.Getenv("DEVBOX_REGION"); region != "" {
		return shellInceptionErrorMsg("devbox cloud shell")
	}

	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}
	return cloud.Shell(cmd.ErrOrStderr(), box.ProjectDir(), flags.githubUsername)
}
