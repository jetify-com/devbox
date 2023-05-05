// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

func installCmd() *cobra.Command {
	flags := runCmdFlags{}
	command := &cobra.Command{
		Use:   "install",
		Short: "Install all packages mentioned in devbox.json",
		Long: "Start a new devbox shell and installs all packages mentioned in devbox.json in current directory or" +
			"a directory specified via --config. \n\n Then exits the shell when packages are done installing.\n\n ",
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
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = box.PrintEnv(cmd.Context(), false /* run init hooks */)
	if err != nil {
		return errors.WithStack(err)
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "Finished installing packages.")
	return nil
}
