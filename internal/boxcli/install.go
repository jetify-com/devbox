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

func installCmd() *cobra.Command {
	flags := runCmdFlags{}
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

	return command
}

func installCmdFunc(cmd *cobra.Command, flags runCmdFlags) error {
	// Check the directory exists.
	box, err := devbox.Open(&devopt.Opts{
		Dir:         flags.config.path,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	if err = box.Install(cmd.Context()); err != nil {
		return errors.WithStack(err)
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "Finished installing packages.")
	return nil
}
