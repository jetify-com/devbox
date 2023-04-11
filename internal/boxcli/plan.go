// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox"
)

type planCmdFlags struct {
	config configFlags
}

func planCmd() *cobra.Command {
	flags := planCmdFlags{}

	command := &cobra.Command{
		Use:    "plan",
		Hidden: true,
		Short:  "Preview the plan used to build your environment",
		Args:   cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanCmd(cmd, flags)
		},
	}

	flags.config.register(command)
	return command
}

func runPlanCmd(cmd *cobra.Command, flags planCmdFlags) error {
	// Check the directory exists.
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}
	enc := json.NewEncoder(cmd.ErrOrStderr())
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	shellPlan, err := box.ShellPlan()
	if err != nil {
		return err
	}

	return errors.WithStack(enc.Encode(shellPlan))
}
