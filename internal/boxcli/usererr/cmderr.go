package usererr

import (
	"errors"
	"fmt"
	"os/exec"
)

// ExecCmdError is an error from an exec.Cmd for a devbox internal command
type ExecCmdError struct {
	err error
	cmd *exec.Cmd
}

func NewExecCmdError(cmd *exec.Cmd, source error) error {
	if source == nil {
		return nil
	}
	return &ExecCmdError{
		cmd: cmd,
		err: source,
	}
}

func (e *ExecCmdError) Error() string {
	var errorMsg string

	// ExitErrors can give us more information so handle that specially.
	var exitErr *exec.ExitError
	if errors.As(e.err, &exitErr) {
		errorMsg = fmt.Sprintf(
			"Error running command %s. Exit status is %d. Command stderr: %s",
			e.cmd, exitErr.ExitCode(), string(exitErr.Stderr),
		)
	} else {
		errorMsg = fmt.Sprintf("Error running command %s. Error: %v", e.cmd, e.err)
	}
	return errorMsg
}

func (e *ExecCmdError) Is(target error) bool {
	return errors.Is(e.err, target)
}

// Unwrap provides compatibility for Go 1.13 error chains.
func (e *ExecCmdError) Unwrap() error { return e.err }
