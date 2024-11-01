// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/ux"
)

type shellCmdFlags struct {
	envFlag
	config       configFlags
	omitNixEnv   bool
	printEnv     bool
	pure         bool
	recomputeEnv bool
}

// shellFlagDefaults are the flag default values that differ
// from the `devbox` command versus `devbox global` command.
type shellFlagDefaults struct {
	omitNixEnv bool
}

func shellCmd(defaults shellFlagDefaults) *cobra.Command {
	flags := shellCmdFlags{}
	command := &cobra.Command{
		Use:   "shell",
		Short: "Start a new shell with access to your packages",
		Long: "Start a new shell with access to your packages.\n\n" +
			"If the --config flag is set, the shell will be started using the devbox.json found in the --config flag directory. " +
			"If --config isn't set, then devbox recursively searches the current directory and its parents.",
		Args:    cobra.NoArgs,
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShellCmd(cmd, flags)
		},
	}

	command.Flags().BoolVar(
		&flags.printEnv, "print-env", false, "print script to setup shell environment")
	command.Flags().BoolVar(
		&flags.pure, "pure", false, "if this flag is specified, devbox creates an isolated shell inheriting almost no variables from the current environment. A few variables, in particular HOME, USER and DISPLAY, are retained.")
	command.Flags().BoolVar(
		&flags.omitNixEnv, "omit-nix-env", defaults.omitNixEnv,
		"shell environment will omit the env-vars from print-dev-env",
	)
	_ = command.Flags().MarkHidden("omit-nix-env")
	command.Flags().BoolVar(&flags.recomputeEnv, "recompute", true, "recompute environment if needed")

	flags.config.register(command)
	flags.envFlag.register(command)
	return command
}

func runShellCmd(cmd *cobra.Command, flags shellCmdFlags) error {
	ctx := cmd.Context()
	env, err := flags.Env(flags.config.path)
	if err != nil {
		return err
	}

	// Check the directory exists.
	box, err := devbox.Open(&devopt.Opts{
		Dir:         flags.config.path,
		Env:         env,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if flags.printEnv {
		// false for includeHooks is because init hooks is not compatible with .envrc files generated
		// by versions older than 0.4.6
		script, err := box.EnvExports(cmd.Context(), devopt.EnvExportsOpts{})
		if err != nil {
			return err
		}
		// explicitly print to stdout instead of stderr so that direnv can read the output
		fmt.Fprint(cmd.OutOrStdout(), script)
		return nil // return here to prevent opening a devbox shell
	}

	if envir.IsDevboxShellEnabled() {
		return shellInceptionErrorMsg("devbox shell")
	}

	onStaleState := func() {
		ux.FHidableWarning(
			ctx,
			cmd.ErrOrStderr(),
			devbox.StateOutOfDateMessage,
			"with --recompute=true",
		)
	}

	return box.Shell(ctx, devopt.EnvOptions{
		Hooks: devopt.EnvLifecycleHooks{
			OnStaleStateWithSkipRecompute: onStaleState,
		},
		OmitNixEnv:    flags.omitNixEnv,
		Pure:          flags.pure,
		SkipRecompute: !flags.recomputeEnv,
	})
}

func shellInceptionErrorMsg(cmdPath string) error {
	return usererr.New("You are already in an active %[1]s.\nRun `exit` before calling `%[1]s` again."+
		" Shell inception is not supported.", cmdPath)
}
