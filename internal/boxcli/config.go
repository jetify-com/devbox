// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/devconfig"
	"go.jetify.com/devbox/internal/envir"
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
		&flags.path, "config", "c", os.Getenv(envir.DevboxConfig), pathFlagUsage,
	)
}

func (flags *pathFlag) registerPersistent(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(
		&flags.path, "config", "c", os.Getenv(envir.DevboxConfig), pathFlagUsage,
	)
}

const pathFlagUsage = "path to directory containing a devbox.json config file " +
	"(defaults to the " + envir.DevboxConfig + " env var, if set)"

// hasProjectConfig reports whether there is a devbox config to open at path,
// or, when path is empty, in the current directory or one of its parents.
// Commands that work with or without a project use it to decide whether to
// open one. Errors other than a missing config report true so that opening
// the project reports them.
func hasProjectConfig(path string) bool {
	var err error
	if path == "" {
		_, err = devconfig.Find(".")
	} else {
		_, err = devconfig.Open(path)
	}
	return !errors.Is(err, devconfig.ErrNotFound)
}
