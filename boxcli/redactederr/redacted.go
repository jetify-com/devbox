package redactederr

import (
	"github.com/pkg/errors"
)

// redacted error implements the error interface
// but excludes Wrap() and Cause() function which expose the original error
// it only preserves the stacktraces of the source error
type redacted struct {
	source error
}

func New(source error) error {
	if source == nil {
		return nil
	}
	return &redacted{
		source: source,
	}
}

func (r *redacted) Error() string {
	// redacted error will not return any original error message.
	return ""
}

// Is uses the source error for comparisons
func (r *redacted) Is(target error) bool {
	return errors.Is(r.source, target)
}

// Preserve stack trace of the source error
func (r *redacted) StackTrace() errors.StackTrace {
	return r.source.(interface { //nolint:errorlint
		StackTrace() errors.StackTrace
	}).StackTrace()
}
