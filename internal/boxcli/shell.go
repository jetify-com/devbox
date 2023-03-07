// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
)

type shellCmdFlags struct {
	config   configFlags
	PrintEnv bool
}

func ShellCmd() *cobra.Command {
	flags := shellCmdFlags{}
	command := &cobra.Command{
		Use:   "shell",
		Short: "Start a new shell with access to your packages",
		Long: "Start a new shell or run a command with access to your packages.\n\n" +
			"If invoked without `cmd`, devbox will start an interactive shell.\n" +
			"If invoked with a `cmd`, devbox will run the command in a shell and then exit.\n" +
			"In both cases, the shell will be started using the devbox.json found in the --config flag directory. " +
			"If --config isn't set, then devbox recursively searches the current directory and its parents.",
		Args:    cobra.NoArgs,
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShellCmd(cmd, args, flags)
		},
	}

	command.Flags().BoolVar(
		&flags.PrintEnv, "print-env", false, "print script to setup shell environment")

	flags.config.register(command)
	return command
}

func runShellCmd(cmd *cobra.Command, args []string, flags shellCmdFlags) error {
	path, _, err := parseShellArgs(cmd, args, flags)
	if err != nil {
		return err
	}
	// Check the directory exists.
	box, err := devbox.Open(path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	if flags.PrintEnv {
		script, err := box.PrintEnv()
		if err != nil {
			return err
		}
		// explicitly print to stdout instead of stderr so that direnv can read the output
		fmt.Fprint(cmd.OutOrStdout(), script)
		return nil // return here to prevent opening a devbox shell
	}

	if devbox.IsDevboxShellEnabled() {
		return shellInceptionErrorMsg("devbox shell")
	}

	return box.Shell()
}

func parseShellArgs(cmd *cobra.Command, args []string, flags shellCmdFlags) (string, []string, error) {
	// TODO: remove code that takes config path from an arguments.
	index := cmd.ArgsLenAtDash()
	if index < 0 {
		configPath, err := configPathFromUser(args, &flags.config)
		if err != nil {
			return "", nil, err
		}
		return configPath, []string{}, nil
	}

	path, err := configPathFromUser(args[:index], &flags.config)
	if err != nil {
		return "", nil, err
	}
	cmds := args[index:]

	return path, cmds, nil
}

func shellInceptionErrorMsg(cmdPath string) error {
	return usererr.New("You are already in an active %[1]s.\nRun `exit` before calling `%[1]s` again."+
		" Shell inception is not supported.", cmdPath)
}
