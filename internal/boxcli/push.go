// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/devbox"
	"go.jetify.com/devbox/internal/devbox/devopt"
)

type pushCmdFlags struct {
	config configFlags
}

func pushCmd() *cobra.Command {
	flags := pushCmdFlags{}
	cmd := &cobra.Command{
		Use:   "push <git-repo>",
		Short: "Push a [global] config to a git repo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pushCmdFunc(cmd, args[0], flags)
		},
	}

	flags.config.register(cmd)

	return cmd
}

func pushCmdFunc(cmd *cobra.Command, url string, flags pushCmdFlags) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:         flags.config.path,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	return box.Push(cmd.Context(), devopt.PullboxOpts{URL: url})
}
