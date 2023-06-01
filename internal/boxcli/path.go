// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/nix"
)

type pathCmdFlags struct {
	config     configFlags
	nixProfile bool
}

func pathCmd() *cobra.Command {
	flags := pathCmdFlags{}
	cmd := &cobra.Command{
		Use:     "path",
		Short:   "Show path to [global] devbox config",
		PreRunE: ensureNixInstalled,
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.nixProfile {
				fmt.Println(filepath.Join(flags.config.path, nix.ProfilePath))
			} else {
				fmt.Println(flags.config.path)
			}
			return nil
		},
	}

	flags.config.register(cmd)
	cmd.Flags().BoolVarP(
		&flags.nixProfile, "nix-profile", "n", false,
		"Show path to nix profile created by devbox",
	)

	return cmd
}
