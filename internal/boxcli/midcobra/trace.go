// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"context"
	"os"
	"runtime/trace"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type TraceMiddleware struct {
	tracef *os.File
	flag   *pflag.Flag
	task   *trace.Task
}

var _ Middleware = (*DebugMiddleware)(nil)

func (t *TraceMiddleware) AttachToFlag(flags *pflag.FlagSet, flagName string) {
	flags.String(flagName, "", "write a trace to a file")
	t.flag = flags.Lookup(flagName)
	t.flag.Hidden = true
	t.flag.NoOptDefVal = "trace.out"
}

func (t *TraceMiddleware) preRun(cmd *cobra.Command, _ []string) {
	if t == nil {
		return
	}
	path := t.flag.Value.String()
	if path == "" {
		return
	}
	var err error
	t.tracef, err = os.Create(path)
	if err != nil {
		panic("error enabling tracing: " + err.Error())
	}
	if err := trace.Start(t.tracef); err != nil {
		panic("error enabling tracing: " + err.Error())
	}

	var ctx context.Context
	ctx, t.task = trace.NewTask(cmd.Context(), "cliCommand")
	cmd.SetContext(ctx)
}

func (t *TraceMiddleware) postRun(*cobra.Command, []string, error) {
	if t.tracef == nil {
		return
	}
	t.task.End()
	trace.Stop()
	if err := t.tracef.Close(); err != nil {
		panic("error closing trace file: " + err.Error())
	}
}
