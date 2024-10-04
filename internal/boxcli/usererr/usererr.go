// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package usererr

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

type level int

const (
	levelError level = iota
	levelWarning
)

type combined struct {
	source      error
	userMessage string
	level       level
	logged      bool
}

// New creates new user error with the given message. By default these errors
// are not logged to Sentry. If you want to log the error, use NewLogged
func New(msg string, args ...any) error {
	return errors.WithStack(&combined{
		userMessage: fmt.Sprintf(msg, args...),
	})
}

// NewLogged creates new user error with the given message. These messages are
// logged to Sentry without the message (for privacy reasons). This is useful
// for unexpected errors that we want to make sure to log but we also want to
// attach a good human readable message to.
func NewLogged(msg string, args ...any) error {
	return errors.WithStack(&combined{
		userMessage: fmt.Sprintf(msg, args...),
		level:       levelError,
		logged:      true,
	})
}

func NewWarning(msg string, args ...any) error {
	return errors.WithStack(&combined{
		userMessage: fmt.Sprintf(msg, args...),
		level:       levelWarning,
	})
}

func WithUserMessage(source error, msg string, args ...any) error {
	// We don't want to wrap the error if it already has a user message. Doing
	// so would obscure the original error message which is likely more useful.
	if source == nil || hasUserMessage(source) {
		return source
	}
	return &combined{
		source:      source,
		userMessage: fmt.Sprintf(msg, args...),
	}
}

func WithLoggedUserMessage(source error, msg string, args ...any) error {
	if source == nil || hasUserMessage(source) {
		return source
	}
	return &combined{
		logged:      true,
		source:      source,
		userMessage: fmt.Sprintf(msg, args...),
	}
}

// Extract unwraps and returns the user error if it exists.
func Extract(err error) (error, bool) { // nolint: revive
	c := &combined{}
	if errors.As(err, &c) {
		return c, true
	}
	return nil, false
}

// ShouldLogError returns true if the it's a combined error specifically marked to be logged
// or if it's not an ExitError.
func ShouldLogError(err error) bool {
	if err == nil {
		return false
	}
	var userExecErr *ExitError
	if errors.As(err, &userExecErr) {
		return false
	}
	c := &combined{}
	if errors.As(err, &c) {
		return c.logged
	}
	return true
}

func IsWarning(err error) bool {
	c := &combined{}
	if errors.As(err, &c) {
		return c.level == levelWarning
	}
	return false
}

func (c *combined) Error() string {
	if c.source == nil {
		return c.userMessage
	}
	return c.userMessage + "\nsource: " + c.source.Error()
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
		_, _ = io.WriteString(s, c.userMessage)
		return
	}
	errors.Wrap(c.source, c.userMessage).(interface { //nolint:errorlint
		Format(s fmt.State, verb rune)
	}).Format(s, verb)
}

func hasUserMessage(err error) bool {
	_, hasUserMessage := Extract(err)
	return hasUserMessage
}
