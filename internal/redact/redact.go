// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package redact implements functions to redact sensitive information from
// errors.
//
// Redacting an error replaces its message with a placeholder while still
// maintaining wrapped errors:
//
//	wrapped := errors.New("not found")
//	name := "Alex"
//	err := fmt.Errorf("error getting user %s: %w", name, wrapped)
//
//	fmt.Println(err)
//	// error getting user Alex: not found
//
//	fmt.Println(Error(err))
//	// <redacted *fmt.wrapError>: <redacted *errors.errorString>
//
// If an error implements a Redact() string method, then it is said to be
// redactable. A redactable error defines an alternative message for its
// redacted form:
//
//	type userErr struct{ name string }
//
//	func (e *userErr) Error() string {
//		return fmt.Sprintf("user %s not found", e.name)
//	}
//
//	func (e *userErr) Redact() string {
//		return fmt.Sprintf("user %x not found", sha256.Sum256([]byte(e.name)))
//	}
//
//	func main() {
//		err := &userErr{name: "Alex"}
//		fmt.Println(err)
//		// user Alex not found
//
//		fmt.Println(Error(err))
//		// user db74c940d447e877d119df613edd2700c4a84cd1cf08beb7cbc319bcfaeab97a not found
//	}
//
// The [Errorf] function creates redactable errors that retain their literal
// format text, but redact any arguments. The format string spec is identical
// to that of [fmt.Errorf]. Calling [Safe] on an [Errorf] argument will include
// it in the redacted message.
//
//	name := "Alex"
//	id := 5
//	err := Errorf("error getting user %s with ID %d", name, Safe(id))
//
//	fmt.Println(err)
//	// error getting user Alex with ID 5
//
//	fmt.Println(Error(err))
//	// error getting user <redacted string> with ID 5
//
//nolint:errorlint
package redact

import (
	"errors"
	"fmt"
	"runtime"
)

// Error returns a redacted error that wraps err. If err has a Redact() string
// method, then Error uses it for the redacted error message. Otherwise, Error
// recursively redacts each wrapped error, joining them with ": " to create the
// final error message. If it encounters an error that has a Redact() method,
// then it appends the result of Redact() to the message and stops unwrapping.
func Error(err error) error {
	if err == nil {
		return nil
	}

	switch t := err.(type) {
	case *redactedError:
		// Don't redact an already redacted error, otherwise its redacted message
		// will be replaced with a placeholder.
		return err
	case redactor:
		return &redactedError{
			msg:     t.Redact(),
			wrapped: err,
		}
	default:
		msg := placeholder(err)
		wrapped := err
		for {
			wrapped = errors.Unwrap(wrapped)
			if wrapped == nil {
				break
			}
			if redactor, ok := wrapped.(redactor); ok {
				msg += ": " + redactor.Redact()
				break
			}
			msg += ": " + placeholder(wrapped)
		}
		return &redactedError{
			msg:     msg,
			wrapped: err,
		}
	}
}

// Errorf creates a redactable error that has an error string identical to that
// of a [fmt.Errorf] error. Calling [Redact] on the result will redact all
// format arguments from the error message instead of redacting the entire
// string.
//
// When redacting the error string, Errorf replaces arguments that implement a
// Redact() string method with the result of that method. To include an
// argument as-is in the redacted error, first call [Safe]. For example:
//
//	username := "bob"
//	Errorf("cannot find user %s", username).Error()
//	// cannot find user <redacted string>
//
//	Errorf("cannot find user %s", Safe(username)).Error()
//	// cannot find user bob
func Errorf(format string, a ...any) error {
	// Capture a stack trace.
	safeErr := &safeError{
		callers: make([]uintptr, 32),
	}
	n := runtime.Callers(2, safeErr.callers)
	safeErr.callers = safeErr.callers[:n]

	// Create the "normal" unredacted error. We need to remove the safe wrapper
	// from any args so that fmt.Errorf can detect and format their type
	// correctly.
	args := make([]any, len(a))
	for i := range a {
		if safe, ok := a[i].(safe); ok {
			args[i] = safe.a
		} else {
			args[i] = a[i]
		}
	}
	safeErr.err = fmt.Errorf(format, args...)

	// Now create the redacted error by replacing all args with their redacted
	// version or by inserting a placeholder if the arg can't be redacted.
	for i := range a {
		switch t := a[i].(type) {
		case safe:
			args[i] = t.a
		case error:
			args[i] = Error(t)
		case redactor:
			args[i] = formatter(t.Redact())
		default:
			args[i] = formatter(placeholder(t))
		}
	}
	safeErr.redacted = fmt.Errorf(format, args...)
	return safeErr
}

// redactor defines the Redact interface for types that can format themselves
// in redacted errors.
type redactor interface {
	Redact() string
}

// safe wraps a value that is marked as safe for including in a redacted error.
type safe struct{ a any }

// Safe marks a value as safe for including in a redacted error.
func Safe(a any) any {
	return safe{a}
}

// safeError is an error that can redact its message.
type safeError struct {
	err      error
	redacted error
	callers  []uintptr
}

func (e *safeError) Error() string  { return e.err.Error() }
func (e *safeError) Redact() string { return e.redacted.Error() }
func (e *safeError) Unwrap() error  { return e.err }

func (e *safeError) StackTrace() []runtime.Frame {
	if len(e.callers) == 0 {
		return nil
	}
	frameIter := runtime.CallersFrames(e.callers)
	frames := make([]runtime.Frame, 0, len(e.callers))
	for {
		frame, more := frameIter.Next()
		frames = append(frames, frame)
		if !more {
			break
		}
	}
	return frames
}

func (e *safeError) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v', 's':
		f.Write([]byte(e.Error()))
		if f.Flag('+') {
			for _, fr := range e.StackTrace() {
				fmt.Fprintf(f, "\n%s\n\t%s:%d", fr.Function, fr.File, fr.Line)
			}
			return
		}
	case 'q':
		fmt.Fprintf(f, "%q", e.Error())
	}
}

// redactedError is an error containing a redacted message. It is usually the
// result of calling Error(safeError).
type redactedError struct {
	msg     string
	wrapped error
}

func (e *redactedError) Error() string { return e.msg }
func (e *redactedError) Unwrap() error { return e.wrapped }

// formatter allows a string to be formatted by any fmt verb.
// For example, fmt.Sprintf("%d", formatter("100")) will return "100" without
// an error.
type formatter string

func (f formatter) Format(s fmt.State, verb rune) {
	s.Write([]byte(f))
}

// placeholder generates a placeholder string for values that don't satisfy
// redactor.
func placeholder(a any) string {
	return fmt.Sprintf("<redacted %T>", a)
}
