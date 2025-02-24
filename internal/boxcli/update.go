// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/boxcli/multi"
	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/devbox"
	"go.jetify.com/devbox/internal/devbox/devopt"
)

type updateCmdFlags struct {
	config      configFlags
	sync        bool
	allProjects bool
	noInstall   bool
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
		"sync all devbox.lock dependencies in multiple projects. "+
			"Dependencies will sync to the latest local version.",
	)
	command.Flags().BoolVar(
		&flags.allProjects,
		"all-projects",
		false,
		"update all projects in the working directory, recursively.",
	)
	command.Flags().BoolVar(
		&flags.noInstall,
		"no-install",
		false,
		"update lockfile but don't install anything",
	)
	return command
}

func updateCmdFunc(cmd *cobra.Command, args []string, flags *updateCmdFlags) error {
	if len(args) > 0 && flags.sync {
		return usererr.New("cannot specify both a package and --sync")
	}

	if flags.allProjects {
		return updateAllProjects(cmd, args)
	}

	if flags.sync {
		return multi.SyncLockfiles(args)
	}

	box, err := devbox.Open(&devopt.Opts{
		Dir:         flags.config.path,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Update(cmd.Context(), devopt.UpdateOpts{
		Pkgs:      args,
		NoInstall: flags.noInstall,
	})
}

func updateAllProjects(cmd *cobra.Command, args []string) error {
	boxes, err := multi.Open(&devopt.Opts{
		Stderr: cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	for _, box := range boxes {
		if err := box.Update(cmd.Context(), devopt.UpdateOpts{
			Pkgs:                  args,
			IgnoreMissingPackages: true,
		}); err != nil {
			return err
		}
	}
	return multi.SyncLockfiles(args)
}
