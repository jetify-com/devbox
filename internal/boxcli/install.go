// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
)

type installCmdFlags struct {
	runCmdFlags
	fixMissingStorePaths bool
}

func installCmd() *cobra.Command {
	flags := installCmdFlags{}
	command := &cobra.Command{
		Use:     "install",
		Short:   "Install all packages mentioned in devbox.json",
		Args:    cobra.MaximumNArgs(0),
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installCmdFunc(cmd, flags)
		},
	}

	flags.config.register(command)
	command.Flags().BoolVar(
		&flags.fixMissingStorePaths, "fix-missing-store-paths", false,
		"Fix missing store paths in the devbox.lock file.",
	)

	return command
}

func installCmdFunc(cmd *cobra.Command, flags installCmdFlags) error {
	// Check the directory exists.
	box, err := devbox.Open(&devopt.Opts{
		Dir:         flags.config.path,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	ctx := cmd.Context()
	if err = box.Install(ctx); err != nil {
		return errors.WithStack(err)
	}
	if flags.fixMissingStorePaths {
		if err = box.FixMissingStorePaths(ctx); err != nil {
			return errors.WithStack(err)
		}
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "Finished installing packages.")
	return nil
}
