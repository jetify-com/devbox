// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package telemetry

import (
	"cmp"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	segment "github.com/segmentio/analytics-go/v3"
	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/nix"

	"go.jetify.com/devbox/internal/build"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/redact"
	"go.jetify.com/devbox/internal/xdg"
)

const appName = "devbox"

type EventName int

const (
	EventCommandSuccess EventName = iota
	EventShellInteractive
	EventShellReady
	EventNixBuildSuccess
)

var (
	deviceID string

	// procStartTime records the start time of the current process.
	procStartTime = time.Now()
	needsFlush    atomic.Bool
	// uploadNow skips upload throttling. It's set when an error is buffered
	// so that errors are reported promptly.
	uploadNow atomic.Bool
	started   bool
)

const (
	// uploadInterval is the minimum time between spawning upload processes.
	// Events are buffered on disk in the meantime.
	uploadInterval = 30 * time.Minute

	// uploadThreshold is the number of buffered events that triggers an
	// upload before uploadInterval has passed.
	uploadThreshold = 50
)

// Start enables telemetry for the current program.
func Start() {
	if started || envir.DoNotTrack() || build.SentryDSN == "" || build.TelemetryKey == "" {
		return
	}

	deviceID = loadDeviceID()
	started = true
}

// Enabled reports whether telemetry was started for the current program.
func Enabled() bool {
	return started
}

// loadDeviceID returns the device ID cached in the state dir, computing and
// caching it if needed. Computing it is slow on macOS because it runs ioreg.
func loadDeviceID() string {
	if data, err := os.ReadFile(deviceIDFile); err == nil {
		if id := strings.TrimSpace(string(data)); id != "" {
			return id
		}
	}

	const deviceSalt = "64ee464f-9450-4b14-8d9c-014c0012ac1a"
	id, err := machineid.ProtectedID(deviceSalt)
	if err != nil {
		return ""
	}
	_ = writeStateFile(deviceIDFile, []byte(id))
	return id
}

func userID() string {
	if username := os.Getenv(envir.GitHubUsername); username != "" {
		const uidSalt = "d6134cd5-347d-4b7c-a2d0-295c0f677948"
		const githubPrefix = "github:"

		// userID is a v5 UUID which is basically a SHA hash of the username.
		// See https://www.uuidtools.com/uuid-versions-explained for a comparison of UUIDs.
		return uuid.NewSHA1(uuid.MustParse(uidSalt), []byte(githubPrefix+username)).String()
	}
	return ""
}

// Stop stops gathering telemetry and, if it's time for an upload, uploads
// buffered events in a separate process.
func Stop() {
	if !started || !needsFlush.Load() {
		return
	}
	started = false
	if !shouldUpload() {
		return
	}

	// Record the upload time before spawning so that other commands running
	// at the same time don't also spawn an uploader.
	_ = writeStateFile(lastUploadFile, []byte(strconv.FormatInt(time.Now().Unix(), 10)))

	// Report errors in a separate process so we don't block exiting.
	exe, err := os.Executable()
	if err == nil {
		_ = exec.Command(exe, "upload-telemetry").Start()
	}
}

// shouldUpload reports whether buffered events should be uploaded now. Uploads
// happen at most once per uploadInterval unless an error was reported or
// there are at least uploadThreshold buffered events.
func shouldUpload() bool {
	if uploadNow.Load() {
		return true
	}
	data, err := os.ReadFile(lastUploadFile)
	if err != nil {
		return true
	}
	lastUpload, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return true
	}
	// A last upload time in the future means the clock changed, so don't
	// trust it.
	if age := time.Since(time.Unix(lastUpload, 0)); age < 0 || age >= uploadInterval {
		return true
	}
	return bufferedEventCount() >= uploadThreshold
}

func bufferedEventCount() int {
	count := 0
	for _, dir := range []string{segmentBufferDir, sentryBufferDir} {
		entries, _ := os.ReadDir(dir)
		count += len(entries)
	}
	return count
}

func Event(e EventName, meta Metadata) {
	if !started {
		return
	}

	switch e {
	case EventCommandSuccess:
		bufferSegmentMessage(commandEvent(meta))
	case EventShellInteractive:
		name := fmt.Sprintf("[%s] Shell Event: interactive", appName)
		msg := newTrackMessage(name, meta)
		bufferSegmentMessage(msg.MessageId, msg)
	case EventShellReady:
		name := fmt.Sprintf("[%s] Shell Event: ready", appName)
		msg := newTrackMessage(name, meta)
		bufferSegmentMessage(msg.MessageId, msg)
	case EventNixBuildSuccess:
		name := fmt.Sprintf("[%s] Nix Build Event: success", appName)
		msg := newTrackMessage(name, meta)
		bufferSegmentMessage(msg.MessageId, msg)
	}
}

func commandEvent(meta Metadata) (id string, msg *segment.Track) {
	name := fmt.Sprintf("[%s] Command: %s", appName, meta.Command)
	msg = newTrackMessage(name, meta)
	return msg.MessageId, msg
}

// Error reports an error to the telemetry server.
func Error(err error, meta Metadata) {
	errToLog := err // use errToLog to avoid shadowing err later. Use err to keep API clean.

	if !started || !usererr.ShouldLogError(errToLog) {
		return
	}

	nixVersion := cmp.Or(nix.Version(), "unknown")

	event := &sentry.Event{
		EventID: sentry.EventID(ExecutionID),
		// Uploads are throttled, so record when the error happened rather
		// than when it's uploaded.
		Timestamp: time.Now(),
		Level:     sentry.LevelError,
		User:      sentry.User{ID: deviceID},
		Exception: newSentryException(redact.Error(errToLog)),
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
			"nix": {
				"version": nixVersion,
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

	// Prefer using the user ID instead of the device ID when it's
	// available.
	if uid := userID(); uid != "" {
		event.User.ID = uid
	}
	bufferSentryEvent(event)
	uploadNow.Store(true)

	msgID, msg := commandEvent(meta)
	msg.Properties["failed"] = true
	msg.Properties["sentry_event_id"] = event.EventID
	bufferSegmentMessage(msgID, msg)
}

type Metadata struct {
	Command      string
	CommandFlags []string
	EventStart   time.Time
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

var (
	sentryBufferDir  = xdg.StateSubpath(filepath.FromSlash("devbox/sentry"))
	segmentBufferDir = xdg.StateSubpath(filepath.FromSlash("devbox/segment"))

	deviceIDFile   = xdg.StateSubpath(filepath.FromSlash("devbox/telemetry/device-id"))
	lastUploadFile = xdg.StateSubpath(filepath.FromSlash("devbox/telemetry/last-upload"))
)

func Upload() {
	wg := sync.WaitGroup{} //nolint:varnamelen
	wg.Add(2)
	go func() {
		defer wg.Done()

		if !initSentryClient(appName) {
			return
		}

		events := restoreEvents[sentry.Event](sentryBufferDir)
		for _, e := range events {
			sentry.CaptureEvent(&e)
		}
		sentry.Flush(3 * time.Second)
	}()
	go func() {
		defer wg.Done()

		if !initSegmentClient() {
			return
		}
		events := restoreEvents[segment.Track](segmentBufferDir)
		for _, e := range events {
			segmentClient.Enqueue(e) //nolint:errcheck
		}
		segmentClient.Close()
	}()
	wg.Wait()
}

func restoreEvents[E any](dir string) []E {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var events []E
	for _, entry := range dirEntries {
		if !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		// Always delete the file so we don't end up with an infinitely growing
		// backlog of errors.
		_ = os.Remove(path)
		if err != nil {
			continue
		}

		var event E
		if err := json.Unmarshal(data, &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events
}

func bufferEvent(file string, event any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	if writeStateFile(file, data) == nil {
		needsFlush.Store(true)
	}
}

// writeStateFile writes data to a file in the XDG state dir, creating its
// parent directory if needed.
func writeStateFile(file string, data []byte) error {
	err := os.WriteFile(file, data, 0o600)
	if errors.Is(err, fs.ErrNotExist) {
		// XDG specifies perms 0700.
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			return err
		}
		err = os.WriteFile(file, data, 0o600)
	}
	return err
}

func newEventID() string {
	// Generate event UUIDs the same way the Sentry SDK does:
	// https://github.com/getsentry/sentry-go/blob/d9ce5344e7e1819921ea4901dd31e47a200de7e0/util.go#L15
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	id[6] &= 0x0F
	id[6] |= 0x40
	id[8] &= 0x3F
	id[8] |= 0x80
	return hex.EncodeToString(id)
}

func ShellStart() time.Time {
	return ParseShellStart(os.Getenv(envir.DevboxShellStartTime))
}

func FormatShellStart(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return strconv.FormatInt(t.Unix(), 10)
}

func ParseShellStart(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	unix, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(unix, 0)
}
