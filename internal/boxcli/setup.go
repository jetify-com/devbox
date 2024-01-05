// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/vercheck"
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
	if err := nix.EnsureNixInstalled(cmd.ErrOrStderr(), nixDaemonFlagVal(cmd)); err != nil {
		return err
	}

	ver, err := nix.Version()
	if err != nil {
		return fmt.Errorf("failed to get nix version: %w", err)
	}

	// ensure minimum nix version installed
	if vercheck.SemverCompare(ver, "2.12.0") < 0 {
		return usererr.New("Devbox requires nix of version >= 2.12. Your version is %s. Please upgrade nix and try again.\n", ver)
	}
	return nil
}

// We return a closure to avoid printing the warning every time and just
// printing it if we actually need the value of the flag.
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
