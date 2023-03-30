// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/nix"
)

const toSearchForPackages = "To search for packages use https://search.nixos.org/packages"

type addCmdFlags struct {
	config configFlags
}

func addCmd() *cobra.Command {
	flags := addCmdFlags{}

	command := &cobra.Command{
		Use:     "add <pkg>...",
		Short:   "Add a new package to your devbox",
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprintf(
					cmd.ErrOrStderr(),
					"Usage: %s\n\n%s\n",
					cmd.UseLine(),
					toSearchForPackages,
				)
				return nil
			}
			err := addCmdFunc(cmd, args, flags)
			if errors.Is(err, nix.ErrPackageNotFound) {
				return usererr.New("%s\n\n%s", err, toSearchForPackages)
			}
			return err
		},
	}

	flags.config.register(command)
	return command
}

func addCmdFunc(cmd *cobra.Command, args []string, flags addCmdFlags) error {
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Add(cmd.Context(), args...)
}
