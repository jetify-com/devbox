// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/impl/devopt"
)

type shellEnvCmdFlags struct {
	envFlag
	config            configFlags
	runInitHook       bool
	install           bool
	pure              bool
	preservePathStack bool
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
			if !strings.HasSuffix(os.Getenv("SHELL"), "fish") {
				fmt.Fprintln(cmd.OutOrStdout(), "hash -r")
			}
			return nil
		},
	}

	command.Flags().BoolVar(
		&flags.runInitHook, "init-hook", false, "runs init hook after exporting shell environment")
	command.Flags().BoolVar(
		&flags.install, "install", false, "install packages before exporting shell environment")

	command.Flags().BoolVar(
		&flags.pure, "pure", false, "If this flag is specified, devbox creates an isolated environment inheriting almost no variables from the current environment. A few variables, in particular HOME, USER and DISPLAY, are retained.")
	command.Flags().BoolVar(
		&flags.preservePathStack, "preserve-path-stack", false,
		"Preserves existing PATH order if this project's environment is already in PATH. "+
			"Useful if you want to avoid overshadowing another devbox project that is already active")
	_ = command.Flags().MarkHidden("preserve-path-stack")

	flags.config.register(command)
	flags.envFlag.register(command)

	command.AddCommand(shellEnvOnlyPathWithoutWrappersCmd())

	return command
}

func shellEnvFunc(cmd *cobra.Command, flags shellEnvCmdFlags) (string, error) {
	env, err := flags.Env(flags.config.path)
	if err != nil {
		return "", err
	}
	box, err := devbox.Open(&devopt.Opts{
		Dir:               flags.config.path,
		Stderr:            cmd.ErrOrStderr(),
		PreservePathStack: flags.preservePathStack,
		Pure:              flags.pure,
		Env:               env,
	})
	if err != nil {
		return "", err
	}

	if flags.install {
		if err := box.Install(cmd.Context()); err != nil {
			return "", err
		}
	}

	envStr, err := box.NixEnv(cmd.Context(), flags.runInitHook)
	if err != nil {
		return "", err
	}

	return envStr, nil
}

func shellEnvOnlyPathWithoutWrappersCmd() *cobra.Command {
	command := &cobra.Command{
		Use:     "only-path-without-wrappers",
		Hidden:  true,
		Short:   "[internal] Print shell command that exports the system $PATH without the bin-wrappers paths.",
		Args:    cobra.ExactArgs(0),
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := shellEnvOnlyPathWithoutWrappersFunc()
			fmt.Fprintln(cmd.OutOrStdout(), s)
			return nil
		},
	}
	return command
}

func shellEnvOnlyPathWithoutWrappersFunc() string {
	return devbox.ExportifySystemPathWithoutWrappers()
}
