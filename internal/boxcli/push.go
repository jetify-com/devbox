// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/goutil"
	"go.jetpack.io/devbox/internal/impl/devopt"
)

type pushCmdFlags struct {
	config configFlags
}

func pushCmd() *cobra.Command {
	flags := pushCmdFlags{}
	cmd := &cobra.Command{
		Use: "push <git-repo>",
		Short: "Push a [global] config. Leave empty to use jetpack cloud. Can " +
			"be a git repo for self storage.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pushCmdFunc(cmd, goutil.GetDefaulted(args, 0), flags)
		},
	}

	flags.config.register(cmd)

	return cmd
}

func pushCmdFunc(cmd *cobra.Command, url string, flags pushCmdFlags) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:    flags.config.path,
		Writer: cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	t, err := genSession()
	var creds devopt.Credentials
	if err != nil {
		return errors.WithStack(err)
	} else if t != nil {
		creds = devopt.Credentials{
			IDToken: t.IDToken,
			Email:   t.IDClaims().Email,
			Sub:     t.IDClaims().ID,
		}
	}
	return box.Push(cmd.Context(), devopt.PullboxOpts{
		URL:         url,
		Credentials: creds,
	})
}
