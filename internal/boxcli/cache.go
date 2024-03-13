// Copyright 2024 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
)

type cacheFlags struct {
	pathFlag
}

func cacheCmd() *cobra.Command {
	flags := cacheFlags{}
	cacheCommand := &cobra.Command{
		Use:               "cache",
		Short:             "Collection of commands to interact with nix cache",
		PersistentPreRunE: ensureNixInstalled,
	}

	copyCommand := &cobra.Command{
		Use:   "copy <uri>",
		Short: "Copies all nix packages in current project to the cache at <uri>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			box, err := devbox.Open(&devopt.Opts{
				Dir:    flags.path,
				Stderr: cmd.ErrOrStderr(),
			})
			if err != nil {
				return errors.WithStack(err)
			}
			return box.CacheCopy(cmd.Context(), args[0])
		},
	}

	flags.pathFlag.register(copyCommand)

	cacheCommand.AddCommand(copyCommand)
	cacheCommand.Hidden = true

	return cacheCommand
}
