// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

type shellEnvCmdFlags struct {
	config      configFlags
	runInitHook bool
}

func shellEnvCmd() *cobra.Command {
	flags := shellEnvCmdFlags{}
	command := &cobra.Command{
		Use:     "shellenv",
		Short:   "Print shell commands that add Devbox packages to your PATH",
		Args:    cobra.ExactArgs(0),
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shellEnvFunc(cmd, flags)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), s)
			return nil
		},
	}

	command.Flags().BoolVar(
		&flags.runInitHook, "init-hook", false, "runs init hook after exporting shell environment")

	flags.config.register(command)
	return command
}

func shellEnvFunc(cmd *cobra.Command, flags shellEnvCmdFlags) (string, error) {
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return "", err
	}

	envStr, err := box.PrintEnv()
	if err != nil {
		return "", err
	}

	if flags.runInitHook {
		initHookStr := box.Config().Shell.InitHook.String()
		return fmt.Sprintf("%s\n%s", envStr, initHookStr), nil
	}

	return envStr, nil
}
