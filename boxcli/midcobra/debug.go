// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.jetpack.io/devbox/debug"
)

type DebugMiddleware struct {
	flag *pflag.Flag
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
	debug.Log("Error: %+v\n", runErr)
}
