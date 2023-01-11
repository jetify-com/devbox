package usererr

import (
	"os/exec"

	"errors"
)

type UserExecError struct {
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
	return &UserExecError{
		err: exitErr,
	}
}

func (e *UserExecError) Error() string {
	return e.err.Error()
}

func (e *UserExecError) Is(target error) bool {
	return errors.Is(e.err, target)
}

func (e *UserExecError) ExitCode() int {
	return e.err.ExitCode()
}

// Unwrap provides compatibility for Go 1.13 error chains.
func (e *UserExecError) Unwrap() error { return e.err }
