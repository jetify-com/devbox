// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/impl/devopt"
)

type servicesCmdFlags struct {
	config configFlags
}

type serviceUpFlags struct {
	background         bool
	processComposeFile string
}

type serviceStopFlags struct {
	allProjects bool
}

func (flags *serviceUpFlags) register(cmd *cobra.Command) {
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

func (flags *serviceStopFlags) register(cmd *cobra.Command) {
	cmd.Flags().BoolVar(
		&flags.allProjects, "all-projects", false, "Stop all running services across all your projects.\nThis flag cannot be used simultaneously with the [services] argument")
}

func servicesCmd() *cobra.Command {
	flags := servicesCmdFlags{}
	serviceUpFlags := serviceUpFlags{}
	serviceStopFlags := serviceStopFlags{}
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
		Short: "Stop one or more services in the current project. If no service is specified, stops all services in the current project.",
		Long:  `Stop one or more services in the current project. If no service is specified, stops all services in the current project. \nIf the --all-projects flag is specified, stops all running services across all your projects. This flag cannot be used with [service] arguments.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopServices(cmd, args, flags, serviceStopFlags)
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
		Use:   "up [service]...",
		Short: "Starts process manager with specified services. If no services are listed, starts the process manager with all the services in your project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return startProcessManager(cmd, args, flags, serviceUpFlags)
		},
	}

	flags.config.registerPersistent(servicesCommand)
	serviceUpFlags.register(upCommand)
	serviceStopFlags.register(stopCommand)
	servicesCommand.AddCommand(lsCommand)
	servicesCommand.AddCommand(upCommand)
	servicesCommand.AddCommand(restartCommand)
	servicesCommand.AddCommand(startCommand)
	servicesCommand.AddCommand(stopCommand)
	return servicesCommand
}

func listServices(cmd *cobra.Command, flags servicesCmdFlags) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:    flags.config.path,
		Writer: cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return box.ListServices(cmd.Context())
}

func startServices(cmd *cobra.Command, services []string, flags servicesCmdFlags) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:    flags.config.path,
		Writer: cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return box.StartServices(cmd.Context(), services...)
}

func stopServices(
	cmd *cobra.Command,
	services []string,
	servicesFlags servicesCmdFlags,
	flags serviceStopFlags,
) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:    servicesFlags.config.path,
		Writer: cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	if len(services) > 0 && flags.allProjects {
		return errors.New("cannot use both services and --all-projects arguments simultaneously")
	}
	return box.StopServices(cmd.Context(), flags.allProjects, services...)
}

func restartServices(
	cmd *cobra.Command,
	services []string,
	flags servicesCmdFlags,
) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:    flags.config.path,
		Writer: cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return box.RestartServices(cmd.Context(), services...)
}

func startProcessManager(
	cmd *cobra.Command,
	args []string,
	servicesFlags servicesCmdFlags,
	flags serviceUpFlags,
) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:    servicesFlags.config.path,
		Writer: cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return box.StartProcessManager(cmd.Context(), args, flags.background, flags.processComposeFile)
}
