// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/vercheck"
)

func selfUpdateCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "self-update",
		Short: "Update devbox launcher and binary",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vercheck.SelfUpdate(cmd.ErrOrStderr())
		},
	}

	return command
}
