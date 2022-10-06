// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/planner/plansdk"
)

func SetupDirenv() *cobra.Command {
	command := &cobra.Command{
		Use:               "setup-direnv",
		Short:             "Prints out a shell script for setting up direnv integration.",
		Args:              cobra.MinimumNArgs(0),
		PersistentPreRunE: nixShellPersistentPreRunE,
		RunE:              setupDirenvFunc(),
	}

	return command
}

func setupDirenvFunc() runFunc {
	return func(cmd *cobra.Command, args []string) error {
		// Note: this path will be changed
		profileDir := ".devbox/profile"
		path := pathArg(args)

		// Check the directory exists.
		box, err := devbox.Open(path, os.Stdout)
		if err != nil {
			return errors.WithStack(err)
		}

		if !plansdk.FileExists(filepath.Join(path, profileDir)) {
			return errors.New("Could not locate the binaries for your devbox project. Run 'devbox shell' and 'exit' to make sure dependencies are installed.")
		}

		return box.SetupDirenv(filepath.Join(path, profileDir, "bin"))
	}
}
