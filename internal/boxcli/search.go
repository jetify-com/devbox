// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/devbox/internal/ux"
)

func searchCmd() *cobra.Command {
	command := &cobra.Command{
		Use:    "search <pkg>",
		Short:  "Search for nix packages",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Fwarning(cmd.ErrOrStderr(), "Search is experimental and may not work as expected.\n\n")
			return searcher.SearchAndPrint(cmd.OutOrStdout(), args[0])
		},
	}

	return command
}
