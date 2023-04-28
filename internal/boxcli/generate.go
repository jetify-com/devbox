// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/cloud"
)

type generateCmdFlags struct {
	config         configFlags
	force          bool
	githubUsername string
}

func generateCmd() *cobra.Command {
	flags := &generateCmdFlags{}

	command := &cobra.Command{
		Use:   "generate",
		Short: "Generate supporting files for your project",
		Args:  cobra.MaximumNArgs(0),
	}
	command.AddCommand(devcontainerCmd())
	command.AddCommand(dockerfileCmd())
	command.AddCommand(debugCmd())
	command.AddCommand(direnvCmd())
	command.AddCommand(sshConfigCmd())
	flags.config.register(command)

	return command
}

func debugCmd() *cobra.Command {
	flags := &generateCmdFlags{}
	command := &cobra.Command{
		Use:    "debug",
		Hidden: true,
		Args:   cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateCmd(cmd, flags)
		},
	}
	return command
}

func devcontainerCmd() *cobra.Command {
	flags := &generateCmdFlags{}
	command := &cobra.Command{
		Use:   "devcontainer",
		Short: "Generate Dockerfile and devcontainer.json files under .devcontainer/ directory",
		Long:  "Generate Dockerfile and devcontainer.json files necessary to run VSCode in remote container environments.",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateCmd(cmd, flags)
		},
	}
	command.Flags().BoolVarP(
		&flags.force, "force", "f", false, "force overwrite on existing files")
	return command
}

func dockerfileCmd() *cobra.Command {
	flags := &generateCmdFlags{}
	command := &cobra.Command{
		Use:   "dockerfile",
		Short: "Generate a Dockerfile that replicates devbox shell",
		Long: "Generate a Dockerfile that replicates devbox shell. " +
			"Can be used to run devbox shell environment in an OCI container.",
		Args: cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateCmd(cmd, flags)
		},
	}
	command.Flags().BoolVarP(
		&flags.force, "force", "f", false, "force overwrite existing files")
	flags.config.register(command)
	return command
}

func direnvCmd() *cobra.Command {
	flags := &generateCmdFlags{}
	command := &cobra.Command{
		Use:   "direnv",
		Short: "Generate a .envrc file that integrates direnv with this devbox project",
		Long: "Generate a .envrc file that integrates direnv with this devbox project. " +
			"Requires direnv to be installed.",
		Args: cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateCmd(cmd, flags)
		},
	}
	command.Flags().BoolVarP(
		&flags.force, "force", "f", false, "force overwrite existing files")
	flags.config.register(command)
	return command
}

func sshConfigCmd() *cobra.Command {
	flags := &generateCmdFlags{}
	command := &cobra.Command{
		Use:    "ssh-config",
		Hidden: true,
		Short:  "Generate ssh config to connect to devbox cloud",
		Long:   "Check ssh config and if they don't exist, it generates the configs necessary to connect to devbox cloud VMs.",
		Args:   cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// ssh-config command is exception and it should run without a config file present
			_, err := cloud.SSHSetup(flags.githubUsername)
			return errors.WithStack(err)
		},
	}
	command.Flags().StringVarP(
		&flags.githubUsername, "username", "u", "", "Github username to use for ssh",
	)
	flags.config.register(command)
	return command
}

func runGenerateCmd(cmd *cobra.Command, flags *generateCmdFlags) error {
	// Check the directory exists.
	box, err := devbox.Open(flags.config.path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}
	switch cmd.Use {
	case "debug":
		return box.Generate()
	case "devcontainer":
		return box.GenerateDevcontainer(flags.force)
	case "dockerfile":
		return box.GenerateDockerfile(flags.force)
	case "direnv":
		return box.GenerateEnvrc(flags.force, "generate")
	}
	return nil
}
