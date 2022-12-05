// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/nix"
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
			return runInstallNixCmd()
		},
	}

	setupCommand.AddCommand(installNixCommand)
	return setupCommand
}

func runInstallNixCmd() error {
	return nix.Install()
}
