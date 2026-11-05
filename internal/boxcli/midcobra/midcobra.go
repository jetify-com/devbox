// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"context"
	"errors"
	"os/exec"

	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/debug"
	"go.jetify.com/devbox/internal/ux"
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
	cmd *cobra.Command

	middlewares []Middleware
}

var _ Executable = (*midcobraExecutable)(nil)

func (ex *midcobraExecutable) AddMiddleware(mids ...Middleware) {
	ex.middlewares = append(ex.middlewares, mids...)
}

func (ex *midcobraExecutable) Execute(ctx context.Context, args []string) int {
	// Ensure cobra uses the same arguments
	ex.cmd.SetContext(ctx)
	// Parse the root's persistent flags (e.g. --debug) so middlewares can see
	// them. Skip unknown flags, which belong to subcommands, instead of
	// stopping at the first one.
	allowlist := ex.cmd.FParseErrWhitelist
	ex.cmd.FParseErrWhitelist.UnknownFlags = true
	_ = ex.cmd.ParseFlags(args)
	ex.cmd.FParseErrWhitelist = allowlist

	// Run the 'pre' hooks
	for _, m := range ex.middlewares {
		m.preRun(ex.cmd, args)
	}

	// set args (needed in case caller transforms args in any way)
	ex.cmd.SetArgs(args)

	// Execute the cobra command:
	err := ex.cmd.Execute()

	// Run the 'post' hooks. Note that unlike the default PostRun cobra functionality these
	// run even if the command resulted in an error. This is useful when we still want to clean up
	// before the program exists or we want to log something. The error, if any, gets passed
	// to the post hook.
	for i := len(ex.middlewares) - 1; i >= 0; i-- {
		ex.middlewares[i].postRun(ex.cmd, args, err)
	}

	if err != nil {
		// If the error is from the exec call, return the exit code of the exec call.
		// Note: order matters! Check if it is a user exec error before a generic exit error.
		var exitErr *exec.ExitError
		var userExecErr *usererr.ExitError
		if errors.As(err, &userExecErr) {
			return userExecErr.ExitCode()
		}
		if errors.As(err, &exitErr) {
			if !debug.IsEnabled() {
				ux.Ferrorf(ex.cmd.ErrOrStderr(), "There was an internal error. "+
					"Run with DEVBOX_DEBUG=1 for a detailed error message, and consider reporting it at "+
					"https://github.com/jetify-com/devbox/issues\n")
			}
			return exitErr.ExitCode()
		}
		return 1 // Error exit code
	}
	return 0
}
