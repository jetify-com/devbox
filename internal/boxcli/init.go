// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/devbox"
)

func initCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "init [<dir>]",
		Short: "Initialize a directory as a devbox project",
		Long: "Initialize a directory as a devbox project. " +
			"This will create an empty devbox.json in the current directory. " +
			"You can then add packages using `devbox add`",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitCmd(args)
		},
	}

	return command
}

func runInitCmd(args []string) error {
	path := pathArg(args)

	_, err := devbox.InitConfig(path)
	return errors.WithStack(err)
}
