// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

func GenerateCmd() *cobra.Command {
	command := &cobra.Command{
		Use:    "generate [<dir>]",
		Args:   cobra.MaximumNArgs(1),
		Hidden: true, // For debugging only
		RunE:   runGenerateCmd,
	}
	return command
}

func runGenerateCmd(cmd *cobra.Command, args []string) error {
	path := pathArg(args)

	// Check the directory exists.
	box, err := devbox.Open(path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Generate()
}
