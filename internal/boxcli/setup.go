// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/nix"
)

func SetupCmd() *cobra.Command {
	setupCommand := &cobra.Command{
		Use:    "setup",
		Short:  "Setup devbox dependencies",
		Hidden: true,
	}

	installNixCommand := &cobra.Command{
		Use:   "nix",
		Short: "Installs Nix",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallNixCmd(cmd)
		},
	}

	setupCommand.AddCommand(installNixCommand)
	return setupCommand
}

func runInstallNixCmd(cmd *cobra.Command) error {
	if nix.NixBinaryInstalled() {
		color.New(color.FgYellow).Fprint(
			cmd.ErrOrStderr(),
			"Nix is already installed. If this is incorrect please remove the "+
				"nix-shell binary from your path.\n",
		)
		return nil
	}
	return nix.Install(cmd.ErrOrStderr())
}
