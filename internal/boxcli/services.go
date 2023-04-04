// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

type servicesCmdFlags struct {
	config configFlags
}

type serviceUpFlags struct {
	configFlags
	background         bool
	processComposeFile string
}

func (flags *serviceUpFlags) register(cmd *cobra.Command) {
	flags.configFlags.register(cmd)
	cmd.Flags().StringVar(
		&flags.processComposeFile,
		"process-compose-file",
		"",
		"path to process compose file or directory containing process "+
			"compose-file.yaml|yml. Default is directory containing devbox.json",
	)
	cmd.Flags().BoolVarP(
		&flags.background, "background", "b", false, "Run service in background")
}

func servicesCmd() *cobra.Command {
	flags := servicesCmdFlags{}
	serviceUpFlags := serviceUpFlags{}
	servicesCommand := &cobra.Command{
		Use:   "services",
		Short: "Interact with devbox services",
	}

	lsCommand := &cobra.Command{
		Use:   "ls",
		Short: "List available services",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return listServices(cmd, flags)
		},
	}

	startCommand := &cobra.Command{
		Use:   "start [service]...",
		Short: "Start service. If no service is specified, starts all services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return startServices(cmd, args, flags)
		},
	}

	stopCommand := &cobra.Command{
		Use:   "stop [service]...",
		Short: "Stop service. If no service is specified, stops all services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopServices(cmd, args, flags)
		},
	}

	restartCommand := &cobra.Command{
		Use:   "restart [service]...",
		Short: "Restart service. If no service is specified, restarts all services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return restartServices(cmd, args, flags)
		},
	}

	upCommand := &cobra.Command{
		Use:   "up",
		Short: "Starts process manager with all supported services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return startProcessManager(cmd, args, serviceUpFlags)
		},
	}

	flags.config.register(servicesCommand)
	serviceUpFlags.register(upCommand)
	servicesCommand.AddCommand(lsCommand)
	servicesCommand.AddCommand(upCommand)
	servicesCommand.AddCommand(restartCommand)
	servicesCommand.AddCommand(startCommand)
	servicesCommand.AddCommand(stopCommand)
	return servicesCommand
}

func listServices(cmd *cobra.Command, flags servicesCmdFlags) error {
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}
	services, err := box.Services()
	if err != nil {
		return err
	}
	for _, service := range services {
		cmd.Println(service.Name)
	}
	return nil
}

func startServices(cmd *cobra.Command, services []string, flags servicesCmdFlags) error {
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	return box.StartServices(cmd.Context(), services...)
}

func stopServices(cmd *cobra.Command, services []string, flags servicesCmdFlags) error {
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}
	return box.StopServices(cmd.Context(), services...)
}

func restartServices(
	cmd *cobra.Command,
	services []string,
	flags servicesCmdFlags,
) error {
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	return box.RestartServices(cmd.Context(), services...)
}

func startProcessManager(cmd *cobra.Command, args []string, flags serviceUpFlags) error {
	box, err := devbox.Open(flags.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	return box.StartProcessManager(cmd.Context(), args, flags.background, flags.processComposeFile)
}
