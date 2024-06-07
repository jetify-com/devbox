// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package debug

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

const DevboxDebug = "DEVBOX_DEBUG"

var (
	level = slog.LevelVar{}
	opts  = slog.HandlerOptions{AddSource: true, Level: &level}
)

func init() {
	enabled, _ := strconv.ParseBool(os.Getenv(DevboxDebug))
	if enabled {
		level.Set(slog.LevelDebug)
	} else {
		// Pick arbitrarily high level to disable all default log levels
		// unless DEVBOX_DEBUG is set.
		level.Set(slog.Level(100))
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &opts)))
}

func Enable()               { level.Set(slog.LevelDebug) }
func IsEnabled() bool       { return slog.Default().Enabled(context.Background(), slog.LevelDebug) }
func SetOutput(w io.Writer) { slog.SetDefault(slog.New(slog.NewTextHandler(w, &opts))) }

func Recover() {
	r := recover()
	if r == nil {
		return
	}

	sentry.CurrentHub().Recover(r)
	if IsEnabled() {
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
