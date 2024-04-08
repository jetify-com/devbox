// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package usererr

import (
	"errors"
	"os/exec"
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
