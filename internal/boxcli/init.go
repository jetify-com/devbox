// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/devbox"
	"go.jetify.com/devbox/internal/ux"
	"go.jetify.com/devbox/pkg/autodetect"
)

type initFlags struct {
	auto   bool
	dryRun bool
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
			err := runInitCmd(cmd, args, flags)
			if errors.Is(err, os.ErrExist) {
				path := pathArg(args)
				if path == "" || path == "." {
					path, _ = os.Getwd()
				}
				ux.Fwarningf(cmd.ErrOrStderr(), "devbox.json already exists in %q.", path)
				err = nil
			}
			return err
		},
	}

	command.Flags().BoolVar(&flags.auto, "auto", false, "Automatically detect packages to add")
	command.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Dry run for auto mode. Prints the config that would be used")
	_ = command.Flags().MarkHidden("auto")
	_ = command.Flags().MarkHidden("dry-run")

	return command
}

func runInitCmd(cmd *cobra.Command, args []string, flags *initFlags) error {
	path := pathArg(args)

	if flags.auto {
		if flags.dryRun {
			config, err := autodetect.DryRun(cmd.Context(), path)
			if err != nil {
				return errors.WithStack(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(config))
			return nil
		}
		return autodetect.InitConfig(cmd.Context(), path)
	}

	return devbox.InitConfig(path)
}
