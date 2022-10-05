// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

func ShellCmd() *cobra.Command {
	command := &cobra.Command{
		Use:               "shell [<dir>] -- [<cmd>]",
		Short:             "Start a new shell or run a command with access to your packages",
		Long:              "Start a new shell or run a command with access to your packages. \nIf invoked without `cmd`, this will start an interactive shell based on the devbox.json in your current directory, or the directory provided with `dir`. \nIf invoked with a `cmd`, this will start a shell based on the devbox.json provided in `dir`, run the command, and then exit.",
		Args:              validateShellArgs,
		PersistentPreRunE: nixShellPersistentPreRunE,
		RunE:              runShellCmd,
	}
	return command
}

func runShellCmd(cmd *cobra.Command, args []string) error {
	path, cmds := parseShellArgs(cmd, args)

	// Check the directory exists.
	box, err := devbox.Open(path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}

	if devbox.IsDevboxShellEnabled() {
		return errors.New("You are already in an active devbox shell.\nRun 'exit' before calling devbox shell again. Shell inception is not supported.")
	}

	if len(cmds) > 0 {
		err = box.Exec(cmds...)
	} else {
		err = box.Shell()
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil
	}
	return err
}

func nixShellPersistentPreRunE(cmd *cobra.Command, args []string) error {
	_, err := exec.LookPath("nix-shell")
	if err != nil {
		return errors.New("could not find nix in your PATH\nInstall nix by following the instructions at https://nixos.org/download.html and make sure you've set up your PATH correctly")
	}
	return nil
}

func validateShellArgs(cmd *cobra.Command, args []string) error {
	lenAtDash := cmd.ArgsLenAtDash()
	if lenAtDash > 1 {
		return fmt.Errorf("accepts at most 1 directory, received %d", lenAtDash)
	}
	return nil
}

func parseShellArgs(cmd *cobra.Command, args []string) (string, []string) {
	index := cmd.ArgsLenAtDash()
	if index < 0 {
		return pathArg(args), []string{}
	}

	path := pathArg(args[:index])
	cmds := args[index:]

	return path, cmds
}
