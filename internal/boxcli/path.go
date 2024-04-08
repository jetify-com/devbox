// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type pathCmdFlags struct {
	config configFlags
}

func pathCmd() *cobra.Command {
	flags := pathCmdFlags{}
	cmd := &cobra.Command{
		Use:     "path",
		Short:   "Show path to [global] devbox config",
		PreRunE: ensureNixInstalled,
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(flags.config.path)
			return nil
		},
	}

	flags.config.register(cmd)

	return cmd
}
