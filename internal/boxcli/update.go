// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/impl/devopt"
)

type updateCmdFlags struct {
	config configFlags
	sync   bool
}

func updateCmd() *cobra.Command {
	flags := &updateCmdFlags{}

	command := &cobra.Command{
		Use:   "update [pkg]...",
		Short: "Update packages in your devbox",
		Long: "Update one, many, or all packages in your devbox. " +
			"If no packages are specified, all packages will be updated. " +
			"Legacy non-versioned packages will be converted to @latest versioned " +
			"packages resolved to their current version.",
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateCmdFunc(cmd, args, flags)
		},
	}

	flags.config.register(command)
	command.Flags().BoolVar(
		&flags.sync,
		"sync-lock",
		false,
		"Sync all devbox.lock dependencies in multiple projects. "+
			"Dependencies will sync to the latest local version.",
	)
	return command
}

func updateCmdFunc(cmd *cobra.Command, args []string, flags *updateCmdFlags) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:    flags.config.path,
		Writer: cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if len(args) > 0 && flags.sync {
		return usererr.New("cannot specify both a package and --sync")
	}

	return box.Update(cmd.Context(), devopt.UpdateOpts{
		Pkgs: args,
		Sync: flags.sync,
	})
}
