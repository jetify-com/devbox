// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/impl/devopt"
)

type shellEnvCmdFlags struct {
	config               configFlags
	runInitHook          bool
	install              bool
	useCachedPrintDevEnv bool
	pure                 bool
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
			fmt.Fprintln(cmd.OutOrStdout(), "hash -r")
			return nil
		},
	}

	command.Flags().BoolVar(
		&flags.runInitHook, "init-hook", false, "runs init hook after exporting shell environment")
	command.Flags().BoolVar(
		&flags.install, "install", false, "install packages before exporting shell environment")

	command.Flags().BoolVar(
		&flags.pure, "pure", false, "If this flag is specified, devbox creates an isolated environment inheriting almost no variables from the current environment. A few variables, in particular HOME, USER and DISPLAY, are retained.")

	// This is no longer used. Remove after 0.4.8 is released.
	command.Flags().BoolVar(
		&flags.useCachedPrintDevEnv,
		"use-cached-print-dev-env",
		false,
		"[internal - not meant for general usage] Use the cached nix print-dev-env environment instead of the current environment",
	)
	// This is used by bin wrappers and not meant for end users.
	command.Flag("use-cached-print-dev-env").Hidden = true
	flags.config.register(command)

	command.AddCommand(shellEnvOnlyPathWithoutWrappersCmd())

	return command
}

func shellEnvFunc(cmd *cobra.Command, flags shellEnvCmdFlags) (string, error) {
	box, err := devbox.Open(&devopt.Opts{
		Dir:    flags.config.path,
		Writer: cmd.ErrOrStderr(),
		Pure:   flags.pure,
	})
	if err != nil {
		return "", err
	}

	if flags.install {
		if err := box.Install(cmd.Context()); err != nil {
			return "", err
		}
	}

	opts := &devopt.PrintEnv{
		Ctx:          cmd.Context(),
		IncludeHooks: flags.runInitHook,
	}
	envStr, err := box.PrintEnv(opts)
	if err != nil {
		return "", err
	}

	return envStr, nil
}

type shellEnvOnlyPathWithoutWrappersCmdFlags struct {
	config configFlags
}

func shellEnvOnlyPathWithoutWrappersCmd() *cobra.Command {
	flags := shellEnvOnlyPathWithoutWrappersCmdFlags{}
	command := &cobra.Command{
		Use:     "only-path-without-wrappers",
		Hidden:  true,
		Short:   "Print shell commands that export PATH without the bin-wrappers",
		Args:    cobra.ExactArgs(0),
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shellEnvOnlyPathWithoutWrappersFunc(cmd, &flags)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), s)
			fmt.Fprintln(cmd.OutOrStdout(), "hash -r")
			return nil
		},
	}
	flags.config.register(command)
	return command
}

func shellEnvOnlyPathWithoutWrappersFunc(cmd *cobra.Command, flags *shellEnvOnlyPathWithoutWrappersCmdFlags) (string, error) {

	box, err := devbox.Open(&devopt.Opts{
		Dir:    flags.config.path,
		Writer: cmd.ErrOrStderr(),
		Pure:   false,
	})
	if err != nil {
		return "", err
	}

	opts := &devopt.PrintEnv{
		Ctx:                     cmd.Context(),
		OnlyPathWithoutWrappers: true,
	}
	envStr, err := box.PrintEnv(opts)
	if err != nil {
		return "", err
	}

	return envStr, nil
}
