// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/nix"
)

type addCmdFlags struct {
	config configFlags
}

func AddCmd() *cobra.Command {
	flags := addCmdFlags{}

	command := &cobra.Command{
		Use:               "add <pkg>...",
		Short:             "Add a new package to your devbox",
		PersistentPreRunE: nix.EnsureInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"Usage: %s\n\nTo search for packages use https://search.nixos.org/packages\n",
					cmd.UseLine(),
				)
				return nil
			}
			return addCmdFunc(cmd, args, flags)
		},
	}

	flags.config.register(command)
	return command
}

func addCmdFunc(_ *cobra.Command, args []string, flags addCmdFlags) error {
	box, err := devbox.Open(flags.config.path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Add(args...)
}
