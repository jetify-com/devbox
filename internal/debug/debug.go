// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package debug

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

const DevboxDebug = "DEVBOX_DEBUG"

var enabled bool

func init() {
	enabled, _ = strconv.ParseBool(os.Getenv(DevboxDebug))
}

func IsEnabled() bool { return enabled }

func Enable() {
	enabled = true
	log.SetPrefix("[DEBUG] ")
	log.SetFlags(log.Llongfile | log.Ldate | log.Ltime)
	_ = log.Output(2, "Debug mode enabled.")
}

func SetOutput(w io.Writer) {
	log.SetOutput(w)
}

func Log(format string, v ...any) {
	if !enabled {
		return
	}
	_ = log.Output(2, fmt.Sprintf(format, v...))
}

func Recover() {
	r := recover()
	if r == nil {
		return
	}

	sentry.CurrentHub().Recover(r)
	if enabled {
		fmt.Fprintln(os.Stderr, "Allowing panic because debug mode is enabled.")
		panic(r)
	}
	fmt.Fprintln(os.Stderr, "Error:", r)
}

func EarliestStackTrace(err error) error {
	type pkgErrorsStackTracer interface{ StackTrace() errors.StackTrace }
	type redactStackTracer interface{ StackTrace() []runtime.Frame }

	var stErr error
	for err != nil {
		//nolint:errorlint
		switch err.(type) {
		case redactStackTracer, pkgErrorsStackTracer:
			stErr = err
		}
		err = errors.Unwrap(err)
	}
	return stErr
}
