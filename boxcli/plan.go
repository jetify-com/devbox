// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

type planCmdFlags struct {
	config configFlags
}

func PlanCmd() *cobra.Command {
	flags := &planCmdFlags{}

	command := &cobra.Command{
		Use:   "plan [<dir>]",
		Short: "Preview the plan used to build your environment",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanCmd(cmd, args, flags)
		},
	}

	registerConfigFlags(command, &flags.config)
	return command
}

func runPlanCmd(cmd *cobra.Command, args []string, flags *planCmdFlags) error {
	path := pathArg(args, &flags.config)

	// Check the directory exists.
	box, err := devbox.Open(path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}

	plan, err := box.BuildPlan()
	if err != nil {
		return errors.WithStack(err)
	}
	if plan.Invalid() {
		return plan.Error()
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return errors.WithStack(enc.Encode(plan))
}
