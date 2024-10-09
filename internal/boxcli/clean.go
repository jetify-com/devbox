// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/devbox"
)

type cleanFlags struct{}

func cleanCmd() *cobra.Command {
	flags := &cleanFlags{}
	command := &cobra.Command{
		Use:   "clean",
		Short: "Cleans up an existing devbox directory.",
		Long: "Cleans up an existing devbox directory." +
			"This will delete all devbox files and directories." +
			"This includes .devbox, devbox.json, devbox.lock.",
		Args: cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCleanCmd(cmd, args, flags)
		},
	}

	return command
}

func runCleanCmd(_ *cobra.Command, args []string, _ *cleanFlags) error {
	path := pathArg(args)
	return devbox.Clean(path)
}
