// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/boxcli/featureflag"
)

type infoCmdFlags struct {
	config configFlags
}

func InfoCmd() *cobra.Command {
	flags := infoCmdFlags{}
	command := &cobra.Command{
		Use:               "info <pkg>",
		Hidden:            !featureflag.PKGConfig.Enabled(),
		Short:             "Display package info",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: nixShellPersistentPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			return infoCmdFunc(cmd, args[0], flags)
		},
	}

	flags.config.register(command)
	return command
}

func infoCmdFunc(_ *cobra.Command, pkg string, flags infoCmdFlags) error {
	box, err := devbox.Open(flags.config.path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Info(pkg)
}
