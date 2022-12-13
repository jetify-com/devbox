// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"go.jetpack.io/devbox/boxcli/usererr"
	"go.jetpack.io/devbox/debug"
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

func (d *DebugMiddleware) preRun(cmd Command, args []string) {
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

func (d *DebugMiddleware) postRun(cmd Command, args []string, runErr error) {
	if runErr == nil {
		return
	}
	if usererr.HasUserMessage(runErr) {
		if usererr.IsWarning(runErr) {
			color.Yellow("\nWarning: %s\n\n", runErr.Error())
		} else {
			color.Red("\nError: " + runErr.Error() + "\n\n")
		}
	} else {
		fmt.Printf("Error: %v\n", runErr)
	}

	st := debug.EarliestStackTrace(runErr)
	debug.Log("Error: %v\nExecutionID:%s\n%+v\n", runErr, d.executionID, st)
}

func (d *DebugMiddleware) withExecutionID(execID string) Middleware {
	d.executionID = execID
	return d
}
