package usererr

import (
	"errors"
	"os/exec"
)

// ExitError is an ExitError for a command run on behalf of a user
type ExitError struct {
	err *exec.ExitError
}

func NewExecError(source error) error {
	if source == nil {
		return nil
	}
	var exitErr *exec.ExitError
	// If the error is not from the exec call, return the original error.
	if !errors.As(source, &exitErr) {
		return source
	}
	return &ExitError{
		err: exitErr,
	}
}

func (e *ExitError) Error() string {
	return e.err.Error()
}

func (e *ExitError) Is(target error) bool {
	return errors.Is(e.err, target)
}

func (e *ExitError) ExitCode() int {
	return e.err.ExitCode()
}

// Unwrap provides compatibility for Go 1.13 error chains.
func (e *ExitError) Unwrap() error { return e.err }
