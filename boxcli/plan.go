// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"
)

type planCmdFlags struct {
	config configFlags
}

func PlanCmd() *cobra.Command {
	flags := planCmdFlags{}

	command := &cobra.Command{
		Use:        "plan",
		Short:      "Preview the plan used to build your environment",
		Deprecated: "Please follow devbox documentation on how to build a container image around your devbox project.",
		Args:       cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	flags.config.register(command)
	return command
}
