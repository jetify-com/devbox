// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/impl/devopt"
	"go.jetpack.io/envsec/pkg/envsec"
	"go.jetpack.io/pkg/envvar"
)

type envsecInitCmdFlags struct {
	config configFlags
	force  bool
}

func envsecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "envsec",
		Short: "envsec commands",
	}
	cmd.AddCommand(envsecInitCmd())
	cmd.Hidden = true
	return cmd
}

func envsecInitCmd() *cobra.Command {
	flags := envsecInitCmdFlags{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "initialize envsec integration",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return envsecInitFunc(cmd, flags)
		},
	}

	flags.config.register(cmd)
	cmd.Flags().BoolVarP(
		&flags.force,
		"force",
		"f",
		false,
		"Force initialization even if already initialized",
	)

	return cmd
}

func envsecInitFunc(cmd *cobra.Command, flags envsecInitCmdFlags) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:    flags.config.path,
		Stderr: cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	if err := defaultEnvsec(cmd).NewProject(cmd.Context(), flags.force); err != nil {
		return errors.WithStack(err)
	}
	box.Config().SetStringField("EnvFrom", "envsec")
	return box.Config().SaveTo(box.ProjectDir())
}

func defaultEnvsec(cmd *cobra.Command) *envsec.Envsec {
	return &envsec.Envsec{
		APIHost: build.JetpackAPIHost(),
		Auth: envsec.AuthConfig{
			ClientID: envvar.Get("ENVSEC_CLIENT_ID", build.ClientID()),
			Issuer:   envvar.Get("ENVSEC_ISSUER", build.Issuer()),
		},
		IsDev:  build.IsDev,
		Stderr: cmd.ErrOrStderr(),
	}
}
