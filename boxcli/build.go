// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/docker"
)

type buildCmdFlags struct {
	config configFlags
	docker docker.BuildFlags
}

func BuildCmd() *cobra.Command {
	flags := buildCmdFlags{}

	command := &cobra.Command{
		Use:        "build",
		Deprecated: "Please follow devbox documentation on how to build a container image around your devbox project.",
		Short:      "(deprecated) Build an OCI image that can run as a container",
		Long:       "(deprecated) Builds your current source directory and devbox configuration as a Docker container. Devbox will create a plan for your container based on your source code, and then apply the packages and stage overrides in your devbox.json. \n To learn more about how to configure your builds, see the [configuration reference](/docs/configuration_reference)",
		Args:       cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	flags.config.register(command)
	command.Flags().StringVar(
		&flags.docker.Name, "name", "devbox", "name for the container")
	command.Flags().BoolVar(
		&flags.docker.NoCache, "no-cache", false, "Do not use a cache")
	command.Flags().StringVar(
		&flags.docker.Engine, "engine", "docker", "Engine used to build the container: 'docker', 'podman'")
	command.Flags().StringSliceVar(
		&flags.docker.Tags, "tags", []string{}, "tags for the container")

	return command
}
