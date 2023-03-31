package telemetry

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	pkgerrors "github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/redact"
)

var ExecutionID string

func init() {
	// Generate event UUIDs the same way the Sentry SDK does:
	// https://github.com/getsentry/sentry-go/blob/d9ce5344e7e1819921ea4901dd31e47a200de7e0/util.go#L15
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	id[6] &= 0x0F
	id[6] |= 0x40
	id[8] &= 0x3F
	id[8] |= 0x80
	ExecutionID = hex.EncodeToString(id)
}

var started bool

// Start enables telemetry for the current program.
func Start(appName string) {
	if appName == "" {
		panic("telemetry.Start: app name is empty")
	}
	if started || DoNotTrack() {
		return
	}

	transport := sentry.NewHTTPTransport()
	transport.Timeout = time.Second * 2
	environment := "production"
	if build.IsDev {
		environment = "development"
	}
	_ = sentry.Init(sentry.ClientOptions{
		Dsn:              build.SentryDSN,
		Environment:      environment,
		Release:          appName + "@" + build.Version,
		Transport:        transport,
		TracesSampleRate: 1,
		BeforeSend: func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
			event.ServerName = "" // redact the hostname, which the SDK automatically adds
			return event
		},
	})
	started = true
}

// Stop stops gathering telemetry and flushes buffered events to the server.
func Stop() {
	if !started {
		return
	}
	sentry.Flush(2 * time.Second)
	started = false
}

type Metadata struct {
	Command      string
	CommandFlags []string
	FeatureFlags map[string]bool

	InShell   bool
	InCloud   bool
	InBrowser bool

	NixpkgsHash string
	Packages    []string

	CloudRegion string
	CloudCache  string
}

func (m *Metadata) cmdContext() map[string]any {
	sentryCtx := map[string]any{}
	if m.Command != "" {
		sentryCtx["Name"] = m.Command
	}
	if len(m.CommandFlags) > 0 {
		sentryCtx["Flags"] = m.CommandFlags
	}
	return sentryCtx
}

func (m *Metadata) envContext() map[string]any {
	sentryCtx := map[string]any{
		"In Shell":   m.InShell,
		"In Cloud":   m.InCloud,
		"In Browser": m.InBrowser,
	}
	if m.CloudCache != "" {
		sentryCtx["Cloud Cache"] = m.CloudCache
	}
	if m.CloudRegion != "" {
		sentryCtx["Cloud Region"] = m.CloudRegion
	}
	return sentryCtx
}

func (m *Metadata) featureContext() map[string]any {
	if len(m.FeatureFlags) == 0 {
		return nil
	}

	sentryCtx := make(map[string]any, len(m.FeatureFlags))
	for name, enabled := range m.FeatureFlags {
		sentryCtx[name] = enabled
	}
	return sentryCtx
}

func (m *Metadata) pkgContext() map[string]any {
	if len(m.Packages) == 0 {
		return nil
	}

	// Every package currently has the same commit hash as its version, but this
	// format will allow us to use individual package versions in the future.
	pkgVersion := "nixpkgs"
	if m.NixpkgsHash != "" {
		pkgVersion += "/" + m.NixpkgsHash
	}
	pkgVersion += "#"
	pkgContext := make(map[string]any, len(m.Packages))
	for _, pkg := range m.Packages {
		pkgContext[pkg] = pkgVersion + pkg
	}
	return pkgContext
}

// Error reports an error to the telemetry server.
func Error(err error, meta Metadata) {
	if !started || err == nil {
		return
	}

	event := &sentry.Event{
		EventID:   sentry.EventID(ExecutionID),
		Level:     sentry.LevelError,
		User:      sentry.User{ID: DeviceID},
		Exception: newSentryException(redact.Error(err)),
		Contexts: map[string]map[string]any{
			"os": {
				"name": build.OS(),
			},
			"device": {
				"arch": runtime.GOARCH,
			},
			"runtime": {
				"name":    "Go",
				"version": strings.TrimPrefix(runtime.Version(), "go"),
			},
		},
	}
	if meta.Command != "" {
		event.Tags = map[string]string{"command": meta.Command}
	}
	if sentryCtx := meta.cmdContext(); len(sentryCtx) > 0 {
		event.Contexts["Command"] = sentryCtx
	}
	if sentryCtx := meta.envContext(); len(sentryCtx) > 0 {
		event.Contexts["Devbox Environment"] = sentryCtx
	}
	if sentryCtx := meta.featureContext(); len(sentryCtx) > 0 {
		event.Contexts["Feature Flags"] = sentryCtx
	}
	if sentryCtx := meta.pkgContext(); len(sentryCtx) > 0 {
		event.Contexts["Devbox Packages"] = sentryCtx
	}
	sentry.CaptureEvent(event)
}

func newSentryException(err error) []sentry.Exception {
	errMsg := err.Error()
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
		if t := exportedErrType(err); t != "" {
			errType = t
		}

		//nolint:errorlint
		switch stackErr := err.(type) {
		// If the error implements the StackTrace method in the redact package, then
		// prefer that. The Sentry SDK gets some things wrong when guessing how
		// to extract the stack trace.
		case interface{ StackTrace() []runtime.Frame }:
			stFunc = stackErr.StackTrace
		// Otherwise use the pkg/errors StackTracer interface.
		case interface{ StackTrace() pkgerrors.StackTrace }:
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
		uw := errors.Unwrap(err)
		if uw == nil {
			break
		}
		err = uw
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
func splitPkgFunc(name string) (pkgPath string, funcName string) {
	// Using the following fully-qualified function name as an example:
	// go.jetpack.io/devbox/internal/impl.(*Devbox).RunScript

	// dir = go.jetpack.io/devbox/internal/
	// base = impl.(*Devbox).RunScript
	dir, base := path.Split(name)

	// pkgName = impl
	// fn = (*Devbox).RunScript
	pkgName, fn, _ := strings.Cut(base, ".")

	// pkgPath = go.jetpack.io/devbox/internal/impl
	// funcName = (*Devbox).RunScript
	return dir + pkgName, fn
}
