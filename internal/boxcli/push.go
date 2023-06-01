// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/pullbox/git"
)

type pushCmdFlags struct {
	config configFlags
	force  bool
}

func pushCmd() *cobra.Command {
	flags := pushCmdFlags{}
	cmd := &cobra.Command{
		Use:     "push <git-repo>",
		Short:   "Push a [global] config to a git repo",
		PreRunE: ensureNixInstalled,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pushCmdFunc(cmd, args[0], flags)
		},
	}

	cmd.Flags().BoolVarP(
		&flags.force, "force", "f", false,
		"Force push even if the remote has diverged",
	)

	flags.config.register(cmd)

	return cmd
}

func pushCmdFunc(cmd *cobra.Command, url string, flags pushCmdFlags) error {
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}
	err = box.Push(url, flags.force)
	if prompt := pushErrorPrompt(err); prompt != "" {
		prompt := &survey.Confirm{Message: prompt}
		if err = survey.AskOne(prompt, &flags.force); err != nil {
			return errors.WithStack(err)
		}
		if !flags.force {
			return nil
		}
		return box.Push(url, flags.force)
	}
	return err
}

func pushErrorPrompt(err error) string {
	switch {
	case errors.Is(err, git.ErrRejected):
		return "Push was rejected. Force?"
	default:
		return ""
	}
}
