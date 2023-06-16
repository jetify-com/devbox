// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/templates"
	"go.jetpack.io/devbox/internal/ux"
)

type createCmdFlags struct {
	showAll  bool
	template string
}

func createCmd() *cobra.Command {
	flags := &createCmdFlags{}
	command := &cobra.Command{
		Use:   "create [dir] --template <template>",
		Short: "Initialize a directory as a devbox project using a template",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.template == "" {
				fmt.Fprintf(
					cmd.ErrOrStderr(),
					"Usage: devbox create [dir] --template <template>\n\n",
				)
				templates.List(cmd.ErrOrStderr(), flags.showAll)
				if !flags.showAll {
					fmt.Fprintf(
						cmd.ErrOrStderr(),
						"\nTo see all available templates, run `devbox create --show-all`\n",
					)
				}
				return nil
			}
			return runCreateCmd(cmd, args, flags)
		},
	}

	command.Flags().StringVarP(
		&flags.template, "template", "t", "",
		"template to initialize the project with",
	)
	command.Flags().BoolVar(
		&flags.showAll, "show-all", false,
		"show all available templates",
	)

	return command
}

func runCreateCmd(cmd *cobra.Command, args []string, flags *createCmdFlags) error {
	path := pathArg(args)
	if path == "" {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, flags.template)
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
