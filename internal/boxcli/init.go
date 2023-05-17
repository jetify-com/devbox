// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/templates"
	"go.jetpack.io/devbox/internal/ux"
)

type initCmdFlags struct {
	template string
}

func initCmd() *cobra.Command {
	flags := &initCmdFlags{}
	command := &cobra.Command{
		Use:   "init [<dir>]",
		Short: "Initialize a directory as a devbox project",
		Long: "Initialize a directory as a devbox project. " +
			"This will create an empty devbox.json in the current directory. " +
			"You can then add packages using `devbox add`",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.template != "" {
				return runTemplateInitCmd(cmd, args, flags)
			}
			return runInitCmd(cmd, args)
		},
	}

	command.Flags().StringVar(
		&flags.template, "template", "", "template to initialize the project with")

	return command
}

func runInitCmd(cmd *cobra.Command, args []string) error {
	path := pathArg(args)

	_, err := devbox.InitConfig(path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func runTemplateInitCmd(
	cmd *cobra.Command,
	args []string,
	flags *initCmdFlags,
) error {
	path := pathArg(args)
	if path == "" {
		path, _ = os.Getwd()
	}

	err := templates.Init(cmd.ErrOrStderr(), flags.template, path)
	if err != nil {
		return err
	}

	ux.Fsuccess(
		cmd.ErrOrStderr(),
		"Initialized devbox project using template %s\n",
		flags.template,
	)

	return nil
}
