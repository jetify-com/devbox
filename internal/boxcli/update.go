// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

type updateCmdFlags struct {
	config configFlags
}

func updateCmd() *cobra.Command {
	flags := &updateCmdFlags{}

	command := &cobra.Command{
		Use:   "update [pkg]...",
		Short: "Updates packages in your devbox",
		Long: "Updates one, many, or all packages in your devbox. " +
			"If no packages are specified, all packages will be updated. " +
			"Only updates versioned packages (e.g. `python@3.11`), not packages that are pinned to a nix channel (e.g. `python3`)",
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateCmdFunc(cmd, args, flags)
		},
	}

	flags.config.register(command)
	return command
}

func updateCmdFunc(
	cmd *cobra.Command,
	args []string,
	flags *updateCmdFlags,
) error {
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Update(cmd.Context(), args...)
}
