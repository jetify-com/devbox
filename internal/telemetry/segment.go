// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package telemetry

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	segment "github.com/segmentio/analytics-go"

	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/cloud/openssh"
	"go.jetpack.io/devbox/internal/env"
)

// cmdStartTime records the time at the start of any devbox command invocation.
var cmdStartTime time.Time

// Event contains common fields used in our segment events
type Event struct {
	AnonymousID string
	AppName     string
	AppVersion  string
	Duration    time.Duration
	CloudRegion string
	OsName      string
	UserID      string
}

type shellAccessKind string

const (
	local   shellAccessKind = "local"
	ssh     shellAccessKind = "ssh"
	browser shellAccessKind = "browser"
)

// NewSegmentClient returns a client object to use for segment logging.
// Callers are responsible for calling client.Close().
func NewSegmentClient(telemetryKey string) segment.Client {
	segmentClient, _ := segment.NewWithConfig(telemetryKey, segment.Config{
		BatchSize: 1, /* no batching */
		// Discard logs:
		Logger:  segment.StdLogger(log.New(io.Discard, "" /* prefix */, 0)),
		Verbose: false,
	})

	return segmentClient
}

// CommandStartTime records and returns the time at the start of the command invocation.
// It must be called initially at the start of the cobra (or other framework) command
// stack. Subsequent calls returns the time from the first invocation of this function.
func CommandStartTime() time.Time {
	if cmdStartTime.IsZero() {
		cmdStartTime = time.Now()
	}
	return cmdStartTime
}

// LogShellDurationEvent logs the duration from start of the command
// till the shell was ready to be interactive.
func LogShellDurationEvent(eventName string, startTime string) error {
	if !Enabled() {
		return nil
	}

	start, err := timeFromUnixTimestamp(startTime)
	if err != nil {
		return errors.WithStack(err)
	}

	evt := Event{
		AnonymousID: DeviceID,
		AppName:     AppDevbox,
		AppVersion:  build.Version,
		CloudRegion: os.Getenv(env.DevboxRegion),
		Duration:    time.Since(start),
		OsName:      build.OS(),
		UserID:      UserIDFromGithubUsername(),
	}

	segmentClient := NewSegmentClient(build.TelemetryKey)
	defer func() {
		_ = segmentClient.Close()
	}()

	// Ignore errors, telemetry is best effort
	_ = segmentClient.Enqueue(segment.Track{
		AnonymousId: evt.AnonymousID,
		// Event name. We trim the prefix from shell-interactive/shell-ready to avoid redundancy.
		Event: fmt.Sprintf("[%s] Shell Event: %s", evt.AppName, strings.TrimPrefix(eventName, "shell-")),
		Context: &segment.Context{
			Device: segment.DeviceInfo{
				Id: evt.AnonymousID,
			},
			App: segment.AppInfo{
				Name:    evt.AppName,
				Version: evt.AppVersion,
			},
			OS: segment.OSInfo{
				Name: evt.OsName,
			},
		},
		Properties: segment.NewProperties().
			Set("shell_access", shellAccess()).
			Set("duration", evt.Duration.Milliseconds()),
		UserId: evt.UserID,
	})
	return nil
}

// UserIDFromGithubUsername returns a uuid string if the user has authenticated with github.
// If not authenticated, or there's an error, then an empty string is returned, which segment
// would treat as logged-out or anonymous user.
func UserIDFromGithubUsername() string {
	username, err := openssh.GithubUsernameFromLocalFile()
	if err != nil || username == "" {
		return ""
	}

	const salt = "d6134cd5-347d-4b7c-a2d0-295c0f677948"
	const githubPrefix = "github:"

	// We use a version 5 uuid.
	// A good comparison of types of uuids is at: https://www.uuidtools.com/uuid-versions-explained
	return uuid.NewSHA1(uuid.MustParse(salt), []byte(githubPrefix+username)).String()
}

// timeFromUnixTimestamp is a helper utility that converts the timestamp string
// into a golang time.Time struct.
//
// See UnixTimestampFromTime for the inverse function.
func timeFromUnixTimestamp(timestamp string) (time.Time, error) {
	i, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Time{}, errors.WithStack(err)
	}
	return time.Unix(i, 0), nil
}

// UnixTimestampFromTime is a helper utility that converts a golang time.Time struct
// to a timestamp string.
//
// See timeFromUnixTimestamp for the inverse function.
func UnixTimestampFromTime(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}

func shellAccess() shellAccessKind {
	// Check if running in devbox cloud
	if env.IsDevboxCloud() {
		// Check if running via ssh tty (i.e. ssh shell)
		if os.Getenv(env.SSHTTY) != "" {
			return ssh
		}
		return browser
	}
	return local
}
