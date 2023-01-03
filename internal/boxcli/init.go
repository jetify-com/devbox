// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/writer"
)

type initCmdFlags struct {
	quiet bool
}

func InitCmd() *cobra.Command {
	flags := initCmdFlags{}
	command := &cobra.Command{
		Use:   "init [<dir>]",
		Short: "Initialize a directory as a devbox project",
		Long:  "Initialize a directory as a devbox project. This will create an empty devbox.json in the current directory. You can then add packages using `devbox add`",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitCmd(cmd, args, flags)
		},
	}
	command.Flags().BoolVarP(
		&flags.quiet, "quiet", "q", false, "Quiet mode: Suppresses logs.")

	return command
}

func runInitCmd(cmd *cobra.Command, args []string, flags initCmdFlags) error {
	path := pathArg(args)

	w := writer.New(cmd)
	_, err := devbox.InitConfig(path, w)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
