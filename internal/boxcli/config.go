// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"
)

// to be composed into xyzCmdFlags structs
type configFlags struct {
	pathFlag
	environment string
}

func (flags *configFlags) register(cmd *cobra.Command) {
	flags.pathFlag.register(cmd)
	cmd.Flags().StringVar(
		&flags.environment, "environment", "dev", "environment to use, when supported (e.g.secrets support dev, prod, preview.)",
	)
}

func (flags *configFlags) registerPersistent(cmd *cobra.Command) {
	flags.pathFlag.registerPersistent(cmd)
	cmd.PersistentFlags().StringVar(
		&flags.environment, "environment", "dev", "environment to use, when supported (e.g. secrets support dev, prod, preview.)",
	)
}

// pathFlag is a flag for specifying the path to a devbox.json file
type pathFlag struct {
	path string
}

func (flags *pathFlag) register(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&flags.path, "config", "c", "", "path to directory containing a devbox.json config file",
	)
}

func (flags *pathFlag) registerPersistent(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(
		&flags.path, "config", "c", "", "path to directory containing a devbox.json config file",
	)
}
