// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

//nolint:errorlint
package redact

import (
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func ExampleError() {
	// Each error string in a chain of wrapped errors is redacted with a
	// placeholder describing the error's type.
	wrapped := errors.New("not found")
	name := "Alex"
	err := fmt.Errorf("error getting user %s: %w", name, wrapped)

	fmt.Println("Normal:", err)
	fmt.Println("Redacted:", Error(err))
	// Output:
	// Normal: error getting user Alex: not found
	// Redacted: <redacted *fmt.wrapError>: <redacted *errors.errorString>
}

func ExampleErrorf() {
	// Errors created with Errorf are redacted by omitting any arguments not
	// marked as safe. The literal portion of the format string is kept.
	wrapped := errors.New("not found")
	name := "Alex"
	id := 5
	err := Errorf("error getting user %s with ID %d: %w", name, Safe(id), wrapped)

	fmt.Println("Normal:", err)
	fmt.Println("Redacted:", Error(err))
	// Output:
	// Normal: error getting user Alex with ID 5: not found
	// Redacted: error getting user <redacted string> with ID 5: <redacted *errors.errorString>
}

func ExampleError_wrapped() {
	// If an error wraps another, then redacting it results in a message with the
	// redacted version of each error in the chain up until the first redactable
	// error.
	name := "Alex"
	err := fmt.Errorf("fatal error: %w",
		Errorf("error getting user %s: %w", name,
			errors.New("not found")))

	fmt.Println("Normal:", err)
	fmt.Println("Redacted:", Error(err))
	// Output:
	// Normal: fatal error: error getting user Alex: not found
	// Redacted: <redacted *fmt.wrapError>: error getting user <redacted string>: <redacted *errors.errorString>
}

func TestNil(t *testing.T) {
	checkUnredactedError(t, nil, "<nil>")
	checkRedactedError(t, nil, "<nil>")
}

func TestSimple(t *testing.T) {
	err := errors.New("simple")
	checkUnredactedError(t, err, "simple")
	checkRedactedError(t, err, "<redacted *errors.errorString>")
}

func TestSimpleWrapSimple(t *testing.T) {
	wrapped := errors.New("error 2")
	err := fmt.Errorf("error 1: %w", wrapped)
	checkUnredactedError(t, err, "error 1: error 2")
	checkRedactedError(t, err, "<redacted *fmt.wrapError>: <redacted *errors.errorString>")
	if !errors.Is(err, wrapped) {
		t.Error("got errors.Is(err, wrapped) == false")
	}
}

func TestRedactor(t *testing.T) {
	err := &testRedactor{msg: "sensitive", redactedMsg: "safe"}
	checkUnredactedError(t, err, "sensitive")
	checkRedactedError(t, err, "safe")
}

func TestRedactorWrapRedactor(t *testing.T) {
	wrapped := &testRedactor{
		msg:         "wrapped sensitive",
		redactedMsg: "wrapped safe",
	}
	err := &testRedactor{
		msg:         "sensitive",
		redactedMsg: "safe",
		err:         wrapped,
	}
	checkUnredactedError(t, err, "sensitive")
	checkRedactedError(t, err, "safe")
	if !errors.Is(err, wrapped) {
		t.Error("got errors.Is(err, wrapped) == false")
	}
}

func TestSimpleWrapRedactor(t *testing.T) {
	wrapped := &testRedactor{
		msg:         "wrapped sensitive",
		redactedMsg: "wrapped safe",
	}
	err := fmt.Errorf("error: %w", wrapped)
	checkUnredactedError(t, err, "error: wrapped sensitive")
	checkRedactedError(t, err, "<redacted *fmt.wrapError>: wrapped safe")
	if !errors.Is(err, wrapped) {
		t.Error("got errors.Is(err, wrapped) == false")
	}
}

func TestNestedWrapRedactor(t *testing.T) {
	nestedWrapped := &testRedactor{
		msg:         "wrapped sensitive",
		redactedMsg: "wrapped safe",
	}
	wrapped := fmt.Errorf("error 2: %w", nestedWrapped)
	err := fmt.Errorf("error 1: %w", wrapped)
	checkUnredactedError(t, err, "error 1: error 2: wrapped sensitive")
	checkRedactedError(t, err, "<redacted *fmt.wrapError>: <redacted *fmt.wrapError>: wrapped safe")
	if !errors.Is(err, wrapped) {
		t.Error("got errors.Is(err, wrapped) == false")
	}
	if !errors.Is(err, nestedWrapped) {
		t.Error("got errors.Is(err, nestedWrapped) == false")
	}
}

func TestErrorf(t *testing.T) {
	err := Errorf("quoted = %q, quotedSafe = %q, int = %d, intSafe = %d",
		"sensitive", Safe("safe"),
		123, Safe(789),
	)
	checkUnredactedError(t, err, `quoted = "sensitive", quotedSafe = "safe", int = 123, intSafe = 789`)
	checkRedactedError(t, err, `quoted = <redacted string>, quotedSafe = "safe", int = <redacted int>, intSafe = 789`)

	// Redact again to check that we don't wipe out the already-redacted message.
	checkRedactedError(t, Error(err), `quoted = <redacted string>, quotedSafe = "safe", int = <redacted int>, intSafe = 789`)
}

func TestErrorfWrapErrorf(t *testing.T) {
	wrapped := Errorf("wrapped string = %s, wrapped safe string = %s", "sensitive", Safe("safe"))
	err := Errorf("error: %w", wrapped)
	checkUnredactedError(t, err, "error: wrapped string = sensitive, wrapped safe string = safe")
	checkRedactedError(t, err, "error: wrapped string = <redacted string>, wrapped safe string = safe")
}

func TestErrorfAs(t *testing.T) {
	wrapped := &customError{
		msg:   "sensitive",
		value: "value",
	}
	err := Errorf("error: %w", wrapped)
	checkUnredactedError(t, err, "error: sensitive")
	checkRedactedError(t, err, "error: <redacted *redact.customError>")

	var unwrapped *customError
	if !errors.As(err, &unwrapped) {
		t.Error("got errors.As(err, unwrapped) == false")
	}
	if unwrapped.value != wrapped.value {
		t.Error("got unwrapped.value != wrapped.value")
	}

	var unwrappedRedacted *customError
	if !errors.As(Error(err), &unwrappedRedacted) {
		t.Error("got errors.As(Error(err), &unwrappedRedacted) == false")
	}
	if unwrappedRedacted.value != wrapped.value {
		t.Error("got unwrappedRedacted.value != wrapped.value")
	}
}

func TestErrorfRedactableArg(t *testing.T) {
	err := Errorf("%d", redactableInt(123))
	checkUnredactedError(t, err, "123")
	checkRedactedError(t, err, "0")
}

func TestErrorFormat(t *testing.T) {
	// Capture the first line of output as the error message and all following
	// lines as the stack trace.
	re := regexp.MustCompile(`(.+)?((?s)
.+/redact.TestErrorFormat
	.+/redact/redact_test.go:\d+
.*
runtime.goexit
	.+:\d+.*)?`)

	cases := []struct {
		format    string
		err       error
		wantMsg   string
		wantStack bool
	}{
		{"%v", Errorf("error %%v"), "error %v", false},
		{"%+v", Errorf("error %%+v"), "error %+v", true},
		{"%s", Errorf("error %%s"), "error %s", false},
		{"%+s", Errorf("error %%+s"), "error %+s", true},
		{"%q", Errorf("error %%q"), `"error %q"`, false},
	}
	for _, test := range cases {
		t.Run(test.format, func(t *testing.T) {
			got := fmt.Sprintf(test.format, test.err)
			groups := re.FindStringSubmatch(got)
			if groups == nil {
				t.Fatal("formatted error doesn't match regexp")
			}
			t.Logf("got formatted stack trace:\n%q", got)
			if got := groups[1]; got != test.wantMsg {
				t.Errorf("got error message %q, want %q", got, test.wantMsg)
			}
			if test.wantStack && (len(groups) < 3 || groups[2] == "") {
				t.Error("got formatted error without stack trace, wanted with stack trace")
			} else if !test.wantStack && len(groups) > 2 && groups[2] != "" {
				t.Error("got formatted error with stack trace, wanted without stack trace")
			}
		})
	}
}

func TestStackTrace(t *testing.T) {
	err := Errorf("error")
	stack := err.(interface{ StackTrace() []runtime.Frame }).StackTrace()
	if len(stack) == 0 {
		t.Fatal("got empty stack trace")
	}
	stackTrace := "got stack trace:\n"
	for _, frame := range stack {
		stackTrace += fmt.Sprintf("%v\n", frame)
	}
	t.Log(stackTrace)

	if !strings.HasSuffix(stack[0].Function, t.Name()) {
		t.Errorf("got stack starting with function name %q, want function ending with test name %q",
			stack[0].Function, t.Name())
	}
	lastFrame := stack[len(stack)-1]
	wantFrame := "runtime.goexit"
	if lastFrame.Function != wantFrame {
		t.Errorf("got stack ending with function name %q, want function name %q",
			lastFrame.Function, wantFrame)
	}
}

func TestMissingStackTrace(t *testing.T) {
	var err interface{ StackTrace() []runtime.Frame } = &safeError{}
	stack := err.StackTrace()
	if stack != nil {
		t.Errorf("got stack with length %d, want nil", len(stack))
	}
}

type testRedactor struct {
	msg         string
	redactedMsg string
	err         error
}

func (e *testRedactor) Error() string  { return e.msg }
func (e *testRedactor) Redact() string { return e.redactedMsg }
func (e *testRedactor) Unwrap() error  { return e.err }

type customError struct {
	msg   string
	value string
}

func (e *customError) Error() string {
	return e.msg
}

type redactableInt int

func (r redactableInt) Redact() string {
	return "0"
}

func checkUnredactedError(t *testing.T, got error, wantMsg string) {
	t.Helper()

	gotMsg := fmt.Sprint(got)
	if gotMsg != wantMsg {
		t.Errorf("got wrong unredacted error:\ngot:  %q\nwant: %q", gotMsg, wantMsg)
	}
}

func checkRedactedError(t *testing.T, got error, wantMsg string) {
	t.Helper()

	gotMsg := fmt.Sprint(Error(got))
	if gotMsg != wantMsg {
		t.Errorf("got wrong redacted error:\ngot:  %q\nwant: %q", gotMsg, wantMsg)
	}
}
