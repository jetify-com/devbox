// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
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
	if nix.BinaryInstalled() {
		color.New(color.FgYellow).Fprint(
			cmd.ErrOrStderr(),
			"Nix is already installed. If this is incorrect please remove the "+
				"nix-shell binary from your path.\n",
		)
		return nil
	}
	return nix.Install(cmd.ErrOrStderr())
}

func ensureNixInstalled(cmd *cobra.Command, args []string) error {
	if nix.BinaryInstalled() {
		return nil
	}
	if nix.DirExists() {
		if err := nix.SourceNixEnv(); err != nil || nix.BinaryInstalled() {
			return err
		}

		return usererr.New(
			"We found a /nix directory but nix binary is not in your PATH and we " +
				"were not able to find it in the usual locations. Your nix installation " +
				"might be broken. If restarting your terminal or reinstalling nix " +
				"doesn't work, please create an issue at " +
				"https://github.com/jetpack-io/devbox/issues",
		)
	}

	color.Yellow("\nNix is not installed. Devbox will attempt to install it.\n\n")

	if isatty.IsTerminal(os.Stdout.Fd()) {
		color.Yellow("Press enter to continue or ctrl-c to exit.\n")
		fmt.Scanln()
	}

	if err := nix.Install(cmd.ErrOrStderr()); err != nil {
		return err
	}

	// Source again
	if err := nix.SourceNixEnv(); err != nil {
		return err
	}

	cmd.PrintErrln("Nix installed successfully. Devbox is ready to use!")
	return nil
}
