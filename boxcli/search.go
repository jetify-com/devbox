// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

func SearchCmd() *cobra.Command {
	command := &cobra.Command{
		Use:               "search <pkg>...",
		Short:             "Search packages in nix store.",
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: nixShellPersistentPreRunE,
		RunE:              searchCmdFunc(),
	}

	return command
}

func searchCmdFunc() runFunc {
	return func(cmd *cobra.Command, args []string) error {
		box, err := devbox.Open(".", os.Stdout)
		if err != nil {
			return errors.WithStack(err)
		}

		return box.Search(args[0])
	}
}
