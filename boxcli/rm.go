// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

type removeCmdFlags struct {
	config configFlags
}

func RemoveCmd() *cobra.Command {
	flags := &removeCmdFlags{}
	command := &cobra.Command{
		Use:   "rm <pkg>...",
		Short: "Remove a package from your devbox",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemoveCmd(cmd, args, flags)
		},
	}

	registerConfigFlags(command, &flags.config)
	return command
}

func runRemoveCmd(_ *cobra.Command, args []string, flags *removeCmdFlags) error {

	dir := ""
	if devbox.IsDevboxShellEnabled() {
		if envdir := os.Getenv("DEVBOX_CONFIG_DIR"); envdir != "" {
			dir = envdir
		}
	}

	box, err := devbox.Open(dir, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Remove(args...)
}
