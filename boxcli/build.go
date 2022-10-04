// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/docker"
)

func BuildCmd() *cobra.Command {
	flags := &docker.BuildFlags{}

	command := &cobra.Command{
		Use:   "build [<dir>]",
		Short: "Build an OCI image that can run as a container",
		Long:  "Builds your current source directory and devbox configuration as a Docker container. Devbox will create a plan for your container based on your source code, and then apply the packages and stage overrides in your devbox.json. \n To learn more about how to configure your builds, see the [configuration reference](/docs/configuration_reference)",
		Args:  cobra.MaximumNArgs(1),
		RunE:  buildCmdFunc(flags),
	}

	command.Flags().StringVar(
		&flags.Name, "name", "devbox", "name for the container")
	command.Flags().BoolVar(
		&flags.NoCache, "no-cache", false, "Do not use a cache")
	command.Flags().StringVar(
		&flags.Engine, "engine", "docker", "Engine used to build the container: 'docker', 'podman'")
	command.Flags().StringSliceVar(
		&flags.Tags, "tags", []string{}, "tags for the container")

	return command
}

func buildCmdFunc(flags *docker.BuildFlags) runFunc {
	return func(cmd *cobra.Command, args []string) error {
		path := pathArg(args)

		// Check the directory exists.
		box, err := devbox.Open(path, os.Stdout)
		if err != nil {
			return errors.WithStack(err)
		}

		return box.Build(flags)
	}
}
