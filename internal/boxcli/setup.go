// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/ux"
)

const nixDaemonFlag = "daemon"

func setupCmd() *cobra.Command {
	setupCommand := &cobra.Command{
		Use:    "setup",
		Short:  "Setup devbox dependencies",
		Hidden: true,
	}

	installNixCommand := &cobra.Command{
		Use:   "nix",
		Short: "Install Nix",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallNixCmd(cmd)
		},
	}

	installNixCommand.Flags().Bool(nixDaemonFlag, false, "Install Nix in multi-user mode.")
	setupCommand.AddCommand(installNixCommand)
	return setupCommand
}

func runInstallNixCmd(cmd *cobra.Command) error {
	if nix.BinaryInstalled() {
		ux.Finfo(
			cmd.ErrOrStderr(),
			"Nix is already installed. If this is incorrect "+
				"please remove the nix-shell binary from your path.\n",
		)
		return nil
	}
	return nix.Install(cmd.ErrOrStderr(), nixDaemonFlagVal(cmd)())
}

// ensureNixInstalled verifies that nix is installed and that it is of a supported version
func ensureNixInstalled(cmd *cobra.Command, _args []string) error {
	return nix.EnsureNixInstalled(cmd.ErrOrStderr(), nixDaemonFlagVal(cmd))
}

// We return a closure to avoid printing the warning every time and just
// printing it if we actually need the value of the flag.
//
// TODO: devbox.Open should run nix.EnsureNixInstalled and do this logic
// internally. Then setup can decide if it wants to pass in the value of the
// nixDaemonFlag (if changed).
func nixDaemonFlagVal(cmd *cobra.Command) func() *bool {
	return func() *bool {
		if !cmd.Flags().Changed(nixDaemonFlag) {
			if os.Geteuid() == 0 {
				ux.Fwarning(
					cmd.ErrOrStderr(),
					"Running as root. Installing Nix in multi-user mode.\n",
				)
				return lo.ToPtr(true)
			}
			return nil
		}

		val, err := cmd.Flags().GetBool(nixDaemonFlag)
		if err != nil {
			return nil
		}
		return &val
	}
}
