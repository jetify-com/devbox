// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"errors"
	"os"
	"os/exec"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/ux"
)

type DebugMiddleware struct {
	executionID string // uuid
	flag        *pflag.Flag
}

var _ Middleware = (*DebugMiddleware)(nil)

func (d *DebugMiddleware) AttachToFlag(flags *pflag.FlagSet, flagName string) {
	flags.Bool(
		flagName,
		false,
		"Show full stack traces on errors",
	)
	d.flag = flags.Lookup(flagName)
	d.flag.Hidden = true
}

func (d *DebugMiddleware) preRun(cmd *cobra.Command, args []string) {
	if d == nil {
		return
	}

	strVal := ""
	if d.flag.Changed {
		strVal = d.flag.Value.String()
	} else {
		strVal = os.Getenv("DEVBOX_DEBUG")
	}
	if enabled, _ := strconv.ParseBool(strVal); enabled {
		debug.Enable()
	}
}

func (d *DebugMiddleware) postRun(cmd *cobra.Command, args []string, runErr error) {
	if runErr == nil {
		return
	}
	if usererr.HasUserMessage(runErr) {
		if usererr.IsWarning(runErr) {
			ux.Fwarning(cmd.ErrOrStderr(), runErr.Error())
		} else {
			color.New(color.FgRed).Fprintf(cmd.ErrOrStderr(), "\nError: %s\n\n", runErr.Error())
		}
	}

	st := debug.EarliestStackTrace(runErr)
	color.New(color.FgRed).Fprintf(cmd.ErrOrStderr(), "Error: %v\n\n", runErr)
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		debug.Log("Command stderr: %s\n", exitErr.Stderr)
	}
	debug.Log("\nExecutionID:%s\n%+v\n", d.executionID, st)
}

func (d *DebugMiddleware) withExecutionID(execID string) Middleware {
	d.executionID = execID
	return d
}
