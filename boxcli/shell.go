// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

func ShellCmd() *cobra.Command {
	command := &cobra.Command{
		Use:  "shell [<dir>]",
		Args: cobra.MaximumNArgs(1),
		RunE: runShellCmd,
	}
	return command
}

func runShellCmd(cmd *cobra.Command, args []string) error {
	path := pathArg(args)

	// Check the directory exists.
	box, err := devbox.Open(path)
	if err != nil {
		return errors.WithStack(err)
	}

	fmt.Println("Installing nix packages. This may take a while...")
	// TODO: If we're inside a devbox shell already, don't re-run.
	return box.Shell()
}
