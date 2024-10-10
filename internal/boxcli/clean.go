// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/devconfig"
)

type cleanFlags struct {
	hard bool
}

const (
	devboxLockFile   = "devbox.lock"
	devboxConfigFile = "devbox.json"
	devboxDotDir     = ".devbox"
)

func cleanCmd() *cobra.Command {
	flags := &cleanFlags{}
	command := &cobra.Command{
		Use:   "clean",
		Short: "Cleans up an existing devbox directory.",
		Long: "Cleans up an existing devbox directory. " +
			"This will delete .devbox and devbox.lock. ",
		Args: cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCleanCmd(cmd, args, flags)
		},
	}

	command.Flags().BoolVar(&flags.hard, "hard", false, "Also delete the devbox.json file")

	return command
}

func runCleanCmd(_ *cobra.Command, args []string, flags *cleanFlags) error {
	path := pathArg(args)

	filesToDelete := []string{
		devboxLockFile,
		devboxDotDir,
	}

	if flags.hard {
		filesToDelete = append(filesToDelete, devboxConfigFile)
	}

	return devconfig.Clean(path, filesToDelete)
}
