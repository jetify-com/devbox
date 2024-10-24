// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/internal/devconfig/configfile"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/shellgen"
)

type cleanFlags struct {
	pathFlag
	hard bool
}

func cleanCmd() *cobra.Command {
	flags := &cleanFlags{}
	command := &cobra.Command{
		Use:   "clean",
		Short: "Clean up devbox files from the current directory",
		Long: "Cleans up an existing devbox directory. " +
			"This will delete .devbox and devbox.lock. ",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.hard {
				prompt := &survey.Confirm{
					Message: "Are you sure you want to delete your devbox config?",
				}
				confirmed := false
				if err := survey.AskOne(prompt, &confirmed); err != nil {
					return err
				}
				if !confirmed {
					return nil
				}
			}
			return runCleanCmd(cmd, args, flags)
		},
	}

	command.Flags().BoolVar(&flags.hard, "hard", false, "Also delete the devbox.json file")

	flags.register(command)

	return command
}

func runCleanCmd(cmd *cobra.Command, args []string, flags *cleanFlags) error {
	path := pathArg(args)

	filesToDelete := []string{
		lock.FileName,
		shellgen.DevboxHiddenDirName,
	}

	if flags.hard {
		filesToDelete = append(filesToDelete, configfile.DefaultName)
	}

	return devconfig.Clean(path, filesToDelete, cmd.ErrOrStderr())
}
