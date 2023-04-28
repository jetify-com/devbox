// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"
)

// to be composed into xyzCmdFlags structs
type configFlags struct {
	path string
}

func (flags *configFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&flags.path, "config", "c", "", "path to directory containing a devbox.json config file",
	)
}
