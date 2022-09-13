// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

func PlanCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "plan [<dir>]",
		Short: "Preview the plan used to build your environment",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runPlanCmd,
	}
	return command
}

func runPlanCmd(cmd *cobra.Command, args []string) error {
	path := pathArg(args)

	// Check the directory exists.
	box, err := devbox.Open(path)
	if err != nil {
		return errors.WithStack(err)
	}

	plan, err := box.Plan()
	if err != nil {
		return errors.WithStack(err)
	}
	if plan.Invalid() {
		return plan.Error()
	}
	fmt.Println(plan)
	return nil
}
