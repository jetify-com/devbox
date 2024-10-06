// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/autodetect"
	"go.jetpack.io/devbox/internal/devbox"
)

type initFlags struct {
	autoDetect bool
	dryRun     bool
}

func initCmd() *cobra.Command {
	flags := &initFlags{}
	command := &cobra.Command{
		Use:   "init [<dir>]",
		Short: "Initialize a directory as a devbox project",
		Long: "Initialize a directory as a devbox project. " +
			"This will create an empty devbox.json in the current directory. " +
			"You can then add packages using `devbox add`",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitCmd(cmd, args, flags)
		},
	}

	command.Flags().BoolVar(&flags.autoDetect, "autodetect", false, "Automatically detect packages to add")
	command.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Dry run")
	command.Flag("autodetect").Hidden = true

	return command
}

func runInitCmd(cmd *cobra.Command, args []string, flags *initFlags) error {
	path := pathArg(args)

	if flags.autoDetect && flags.dryRun {
		return autodetect.DryRun(cmd.Context(), path, cmd.ErrOrStderr())
	}

	err := devbox.InitConfig(path)
	if err != nil {
		return errors.WithStack(err)
	}

	if flags.autoDetect {
		err = autodetect.PopulateConfig(cmd.Context(), path, cmd.ErrOrStderr())
	}

	return errors.WithStack(err)
}
