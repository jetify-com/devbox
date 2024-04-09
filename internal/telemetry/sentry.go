// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package telemetry

import (
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/build"
)

var ExecutionID = newEventID()

func initSentryClient(appName string) bool {
	if appName == "" {
		panic("telemetry.Start: app name is empty")
	}
	if build.SentryDSN == "" {
		return false
	}

	transport := sentry.NewHTTPTransport()
	transport.Timeout = time.Second * 2
	environment := "production"
	if build.IsDev {
		environment = "development"
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              build.SentryDSN,
		Environment:      environment,
		Release:          appName + "@" + build.Version,
		Transport:        transport,
		TracesSampleRate: 1,
		BeforeSend: func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
			// redact the hostname, which the SDK automatically adds
			event.ServerName = ""
			return event
		},
	})
	return err == nil
}

func newSentryException(errToLog error) []sentry.Exception {
	errMsg := errToLog.Error()
	binPkg := ""
	modPath := ""
	if build, ok := debug.ReadBuildInfo(); ok {
		binPkg = build.Path
		modPath = build.Main.Path
	}

	// Unwrap in a loop to get the most recent stack trace. stFunc is set to a
	// function that can generate a stack trace for the most recent error. This
	// avoids computing the full stack trace for every error.
	var stFunc func() []runtime.Frame
	errType := "Generic Error"
	for {
		if t := exportedErrType(errToLog); t != "" {
			errType = t
		}

		//nolint:errorlint
		switch stackErr := errToLog.(type) {
		// If the error implements the StackTrace method in the redact package, then
		// prefer that. The Sentry SDK gets some things wrong when guessing how
		// to extract the stack trace.
		case interface{ StackTrace() []runtime.Frame }:
			stFunc = stackErr.StackTrace
		// Otherwise use the pkg/errors StackTracer interface.
		case interface{ StackTrace() errors.StackTrace }:
			// Normalize the pkgs/errors.StackTrace type to a slice of runtime.Frame.
			stFunc = func() []runtime.Frame {
				pkgStack := stackErr.StackTrace()
				pc := make([]uintptr, len(pkgStack))
				for i := range pkgStack {
					pc[i] = uintptr(pkgStack[i])
				}
				frameIter := runtime.CallersFrames(pc)
				frames := make([]runtime.Frame, 0, len(pc))
				for {
					frame, more := frameIter.Next()
					frames = append(frames, frame)
					if !more {
						break
					}
				}
				return frames
			}
		}
		uw := errors.Unwrap(errToLog)
		if uw == nil {
			break
		}
		errToLog = uw
	}
	ex := []sentry.Exception{{Type: errType, Value: errMsg}}
	if stFunc != nil {
		ex[0].Stacktrace = newSentryStack(stFunc(), binPkg, modPath)
	}
	return ex
}

func newSentryStack(frames []runtime.Frame, binPkg, modPath string) *sentry.Stacktrace {
	stack := &sentry.Stacktrace{
		Frames: make([]sentry.Frame, len(frames)),
	}
	for i, frame := range frames {
		pkgName, funcName := splitPkgFunc(frame.Function)

		// The entrypoint has the full function name "main.main". Replace the
		// package name with its full package path to make it easier to find.
		if pkgName == "main" {
			pkgName = binPkg
		}

		// The file path will be absolute unless the binary was built with -trimpath
		// (which releases should be). Absolute paths make it more difficult for
		// Sentry to correctly group errors, but there's no way to infer a relative
		// path from an absolute path at runtime.
		var absPath, relPath string
		if filepath.IsAbs(frame.File) {
			absPath = frame.File
		} else {
			relPath = frame.File
		}

		// Reverse the frames - Sentry wants the most recent call first.
		stack.Frames[len(frames)-i-1] = sentry.Frame{
			Function: funcName,
			Module:   pkgName,
			Filename: relPath,
			AbsPath:  absPath,
			Lineno:   frame.Line,
			InApp:    strings.HasPrefix(frame.Function, modPath) || pkgName == binPkg,
		}
	}
	return stack
}

// exportedErrType returns the underlying type name of err if it's exported.
// Otherwise, it returns an empty string.
func exportedErrType(err error) string {
	t := reflect.TypeOf(err)
	if t == nil {
		return ""
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name := t.Name()
	if r, _ := utf8.DecodeRuneInString(name); unicode.IsUpper(r) {
		return t.String()
	}
	return ""
}

// splitPkgFunc splits a fully-qualified function or method name into its
// package path and base name components.
func splitPkgFunc(name string) (pkgPath, funcName string) {
	// Using the following fully-qualified function name as an example:
	// go.jetpack.io/devbox/internal/devbox.(*Devbox).RunScript

	// dir = go.jetpack.io/devbox/internal/
	// base = devbox.(*Devbox).RunScript
	dir, base := path.Split(name)

	// pkgName = devbox
	// fn = (*Devbox).RunScript
	pkgName, fn, _ := strings.Cut(base, ".")

	// pkgPath = go.jetpack.io/devbox/internal/devbox
	// funcName = (*Devbox).RunScript
	return dir + pkgName, fn
}

// bufferSentryEvent buffers a Sentry event to disk so that Report can upload it
// later.
func bufferSentryEvent(event *sentry.Event) {
	bufferEvent(filepath.Join(sentryBufferDir, string(event.EventID)+".json"), event)
}
