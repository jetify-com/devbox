package usererr

import (
	"fmt"

	"github.com/pkg/errors"
)

type combined struct {
	source      error
	userMessage string
}

func New(msg string, args ...any) error {
	return errors.WithStack(&combined{
		userMessage: fmt.Sprintf(msg, args...),
	})
}

func WithUserMessage(source error, msg string, args ...any) error {
	if source == nil {
		return nil
	}
	return &combined{
		source:      source,
		userMessage: fmt.Sprintf(msg, args...),
	}
}

func HasUserMessage(err error) bool {
	c := &combined{}
	return errors.As(err, &c) // note double pointer
}

func (c *combined) Error() string {
	if c.source == nil {
		return c.userMessage
	}
	return c.userMessage + ": " + c.source.Error()
}

// Is uses the source error for comparisons
func (c *combined) Is(target error) bool {
	return errors.Is(c.source, target)
}

// Unwrap provides compatibility for Go 1.13 error chains.
func (c *combined) Unwrap() error { return c.Cause() }

// Leverage functionality of errors.Cause
func (c *combined) Cause() error { return errors.Cause(c.source) }

// Format allows us to use %+v as implemented by github.com/pkg/errors.
func (c *combined) Format(s fmt.State, verb rune) {
	if c.source == nil {
		fmt.Fprintf(s, c.userMessage)
		return
	}
	errors.Wrap(c.source, c.userMessage).(interface { //nolint:errorlint
		Format(s fmt.State, verb rune)
	}).Format(s, verb)
}
