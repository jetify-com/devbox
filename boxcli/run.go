// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

type runCmdFlags struct {
	config configFlags
}

func RunCmd() *cobra.Command {
	flags := runCmdFlags{}
	command := &cobra.Command{
		Use:               "run -- [<target>]",
		Short:             "Starts a new interactive shell running your target task. The shell will exit once your target task is completed or when it is terminated via CTRL-C",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: nixShellPersistentPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskCmd(cmd, args, flags)
		},
	}

	flags.config.register(command)

	return command
}

func runTaskCmd(cmd *cobra.Command, args []string, flags runCmdFlags) error {
	path, task, err := parseTaskArgs(args, flags)
	if err != nil {
		return err
	}

	// Check the directory exists.
	box, err := devbox.Open(path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}

	if devbox.IsDevboxShellEnabled() {
		return errors.New("You are already in an active devbox shell.\nRun 'exit' before calling devbox shell again. Shell inception is not supported.")
	}

	// For now -- pass this task exactly one target, and let it execute
	err = box.RunTask(task)

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil
	}
	return err
}

func parseTaskArgs(args []string, flags runCmdFlags) (string, string, error) {
	path, err := configPathFromUser([]string{}, &flags.config)
	if err != nil {
		return "", "", err
	}

	task := args[0]

	return path, task, nil
}
