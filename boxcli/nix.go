// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/nix"
)

func NixCmd() *cobra.Command {
	nixCommand := &cobra.Command{
		Use:    "nix",
		Short:  "Commands that interface with Nix",
		Hidden: true,
	}

	installCommand := &cobra.Command{
		Use:   "install",
		Short: "Installs Nix",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallNixCmd()
		},
	}

	nixCommand.AddCommand(installCommand)
	return nixCommand
}

func runInstallNixCmd() error {
	return nix.Install()
}
