// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"context"

	"github.com/spf13/cobra"
)

type Executable interface {
	AddMiddleware(mids ...Middleware)
	Execute(ctx context.Context, args []string) int
}

type Middleware interface {
	preRun(cmd *cobra.Command, args []string)
	postRun(cmd *cobra.Command, args []string, runErr error)
}

func New(cmd *cobra.Command) Executable {
	return &midcobraExecutable{
		cmd:         cmd,
		middlewares: []Middleware{},
	}
}

type midcobraExecutable struct {
	cmd         *cobra.Command
	middlewares []Middleware
}

var _ Executable = (*midcobraExecutable)(nil)

func (ex *midcobraExecutable) AddMiddleware(mids ...Middleware) {
	ex.middlewares = append(ex.middlewares, mids...)
}

func (ex *midcobraExecutable) Execute(ctx context.Context, args []string) int {
	// Ensure cobra uses the same arguments
	ex.cmd.SetArgs(args)

	// Run the 'pre' hooks
	for _, m := range ex.middlewares {
		m.preRun(ex.cmd, args)
	}

	// Execute the cobra command:
	err := ex.cmd.ExecuteContext(ctx)

	// Run the 'post' hooks. Note that unlike the default PostRun cobra functionality these
	// run even if the command resulted in an error. This is useful when we still want to clean up
	// before the program exists or we want to log something. The error, if any, gets passed
	// to the post hook.
	for _, m := range ex.middlewares {
		m.postRun(ex.cmd, args, err)
	}

	if err != nil {
		return 1 // Error exit code
	} else {
		return 0
	}
}
