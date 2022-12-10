// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/boxcli/featureflag"
)

type servicesCmdFlags struct {
	config configFlags
}

func ServicesCmd() *cobra.Command {
	flags := servicesCmdFlags{}
	servicesCommand := &cobra.Command{
		Use:    "services",
		Hidden: !featureflag.PKGConfig.Enabled(),
		Short:  "Interact with devbox services",
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
		Use:   "start [service]",
		Short: "Starts service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return startService(args[0], flags)
		},
	}

	stopCommand := &cobra.Command{
		Use:   "stop [service]",
		Short: "Stops service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopService(args[0], flags)
		},
	}

	flags.config.register(servicesCommand)
	servicesCommand.AddCommand(lsCommand)
	servicesCommand.AddCommand(startCommand)
	servicesCommand.AddCommand(stopCommand)
	return servicesCommand
}

func listServices(cmd *cobra.Command, flags servicesCmdFlags) error {
	box, err := devbox.Open(flags.config.path, os.Stdout)
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

func startService(service string, flags servicesCmdFlags) error {
	box, err := devbox.Open(flags.config.path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}
	return box.StartService(service)
}

func stopService(service string, flags servicesCmdFlags) error {
	box, err := devbox.Open(flags.config.path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}
	return box.StopService(service)
}
