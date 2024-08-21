// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/templates"
	"go.jetpack.io/devbox/internal/ux"
)

type createCmdFlags struct {
	showAll  bool
	template string
	repo     string
	subdir   string
}

func createCmd() *cobra.Command {
	flags := &createCmdFlags{}
	command := &cobra.Command{
		Use:   "create [dir] --template <template>",
		Short: "Initialize a directory as a devbox project using a template",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.template == "" && flags.repo == "" {
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
	command.Flags().StringVarP(
		&flags.repo, "repo", "r", "",
		"Git repository HTTPS URL to import template files from. Example: https://github.com/jetify-com/devbox",
	)
	command.Flags().StringVarP(
		&flags.subdir, "subdir", "s", "",
		"Subdirectory of the Git repository in which the template files reside. Example: examples/tutorial",
	)
	// this command marks a flag as hidden. Error handling for it is not necessary.
	_ = command.Flags().MarkHidden("repo")
	_ = command.Flags().MarkHidden("subdir")

	return command
}

func runCreateCmd(cmd *cobra.Command, args []string, flags *createCmdFlags) error {
	path := handlePath(args, flags)

	var err error
	if flags.template != "" {
		err = templates.InitFromName(cmd.ErrOrStderr(), flags.template, path)
	} else if flags.repo != "" {
		err = templates.InitFromRepo(cmd.ErrOrStderr(), flags.repo, flags.subdir, path)
	} else {
		err = usererr.New("either --template or --repo need to be specified")
	}
	if err != nil {
		return err
	}

	ux.Fsuccessf(
		cmd.ErrOrStderr(),
		"Initialized devbox project using template %s\n",
		flags.template,
	)

	return nil
}

func handlePath(args []string, flags *createCmdFlags) string {
	path := pathArg(args)
	wd, _ := os.Getwd()
	if path == "" {
		if flags.template != "" {
			path = filepath.Join(wd, flags.template)
		} else if flags.repo != "" && flags.subdir == "" {
			path = filepath.Join(wd, filepath.Base(flags.repo))
		} else if flags.repo != "" && flags.subdir != "" {
			path = filepath.Join(wd, filepath.Base(flags.subdir))
		}
	}
	return path
}
