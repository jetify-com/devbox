// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package usererr

import (
	"errors"
	"os/exec"
	"syscall"
)

// ExitError is an ExitError for a command run on behalf of a user
type ExitError struct {
	*exec.ExitError
}

func NewExecError(source error) error {
	if source == nil {
		return nil
	}

	// BUG(gcurtis): exec.Cmd.Run can return other error types, such as when the
	// binary path isn't found. Those should still be considered a user exec error
	// and not reported to Sentry.
	var exitErr *exec.ExitError
	if !errors.As(source, &exitErr) {
		return source
	}
	return &ExitError{ExitError: exitErr}
}

// ExitCode returns the command's exit code or, if a signal killed it, 128 plus
// the signal number, the same as a shell reports it (e.g. 130 for Ctrl-C).
func (e *ExitError) ExitCode() int {
	if status, ok := e.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return e.ExitError.ExitCode()
}

// Interrupted reports whether the command was killed by SIGINT, which usually
// means the user pressed Ctrl-C.
func (e *ExitError) Interrupted() bool {
	status, ok := e.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && status.Signal() == syscall.SIGINT
}
