// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/devbox"
)

type cleanFlags struct {
	pathFlag
}

func cleanCmd() *cobra.Command {
	flags := cleanFlags{}
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up devbox files from the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := &survey.Confirm{
				Message: "Are you sure you want to clean up devbox files?",
			}
			confirmed := false
			if err := survey.AskOne(prompt, &confirmed); err != nil {
				return err
			}
			if !confirmed {
				return nil
			}
			return devbox.Clean(flags.path, cmd.ErrOrStderr())
		},
	}
	flags.register(cmd)
	return cmd
}
