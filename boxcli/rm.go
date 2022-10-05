// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

func RemoveCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "rm <pkg>...",
		Short: "Remove a package from your devbox",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runRemoveCmd,
	}
	return command
}

func runRemoveCmd(cmd *cobra.Command, args []string) error {
	box, err := devbox.Open(".", os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Remove(args...)
}
