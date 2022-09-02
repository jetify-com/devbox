// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

func (d *DebugMiddleware) preRun(cmd *cobra.Command, args []string) {}

func (d *DebugMiddleware) postRun(cmd *cobra.Command, args []string, runErr error) {
	if runErr != nil && d.Debug() {
		log.Printf("Error: %+v\n", runErr)
	}
}

func (d *DebugMiddleware) Debug() bool {
	if d != nil && d.flag.Changed {
		return d.flag.Value.String() == "true"
	}
	return os.Getenv("DEBUG") != ""
}
