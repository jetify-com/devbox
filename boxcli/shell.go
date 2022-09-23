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
		Short:             "Start a new shell with access to your packages",
		Args:              validateShellArgs,
		PersistentPreRunE: nixShellPersistentPreRunE,
		RunE:              runShellCmd,
	}
	return command
}

func runShellCmd(cmd *cobra.Command, args []string) error {
	path, cmds := parseShellArgs(cmd, args)

	// Check the directory exists.
	box, err := devbox.Open(path)
	if err != nil {
		return errors.WithStack(err)
	}

	inDevboxShell := os.Getenv("DEVBOX_SHELL_ENABLED")
	if inDevboxShell != "" && inDevboxShell != "0" && inDevboxShell != "false" {
		return errors.New("You are already in an active devbox shell.\nRun 'exit' before calling devbox shell again. Shell inception is not supported.")
	}

	fmt.Println("Installing nix packages. This may take a while...")

	if len(cmds) > 0 {
		err = box.Exec(cmds...)
	} else {
		err = box.Shell()
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
		index = 0
	}

	path := pathArg(args[:index])
	cmds := args[index:]
	return path, cmds
}
