// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/pullbox/git"
)

type pushCmdFlags struct {
	config configFlags
	force  bool
}

func pushCmd() *cobra.Command {
	flags := pushCmdFlags{}
	cmd := &cobra.Command{
		Use:     "push",
		Short:   "Push a config to a git repo",
		Long:    "Push a config to a git repo. This will create a commit if needed and push it to the remote.",
		PreRunE: ensureNixInstalled,
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pushCmdFunc(cmd, flags)
		},
	}

	cmd.Flags().BoolVarP(
		&flags.force, "force", "f", false,
		"Force push even if the remote has diverged",
	)

	flags.config.register(cmd)

	return cmd
}

func pushCmdFunc(cmd *cobra.Command, flags pushCmdFlags) error {
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}
	err = box.Push(flags.force)
	if prompt := pushErrorPrompt(err); prompt != "" {
		prompt := &survey.Confirm{Message: prompt}
		if err = survey.AskOne(prompt, &flags.force); err != nil {
			return errors.WithStack(err)
		}
		if !flags.force {
			return nil
		}
		return box.Push(flags.force)
	}
	if errors.Is(err, git.ErrNotAGitRepo) {
		return usererr.New(
			"Not a git repo. Use 'devbox [global] pull <repo>' to follow"+
				" a repo.\nYou can also cd to %s and manually set up a git repo.",
			flags.config.path,
		)
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
