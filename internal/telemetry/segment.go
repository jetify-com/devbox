package telemetry

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	segment "github.com/segmentio/analytics-go"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/cloud/openssh"
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

// ttiEvent contains fields used to log the time-to-interactive event. For now,
// this is used for devbox shell (local and cloud).
type ttiEvent struct {
	Event
	eventName string
}

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
func LogShellDurationEvent(eventName string, startTime string, sentrySpan string) error {
	opts := InitOpts()
	if IsDisabled(opts) {
		// disabled
		return nil
	}

	fmt.Printf("sentrySpan: %s\n", sentrySpan)
	var spanStruct struct {
		*sentry.Span
		ParentSpanID string `json:"parent_span_id,omitempty"`
	}
	if err := json.Unmarshal([]byte(sentrySpan), &spanStruct); err != nil {
		return errors.WithStack(err)
	}

	spanStruct.Finish()

	start, err := timeFromUnixTimestamp(startTime)
	if err != nil {
		return errors.WithStack(err)
	}

	evt := ttiEvent{
		Event: Event{
			AnonymousID: DeviceID(),
			AppName:     opts.AppName,
			AppVersion:  opts.AppVersion,
			CloudRegion: os.Getenv("DEVBOX_REGION"),
			Duration:    time.Since(start),
			OsName:      OS(),
			UserID:      UserIDFromGithubUsername(),
		},
		eventName: eventName,
	}

	segmentClient := NewSegmentClient(build.TelemetryKey)
	defer func() {
		_ = segmentClient.Close()
	}()

	fmt.Printf("For event %s, duration: %s\n", evt.eventName, evt.Duration.Milliseconds())

	// Ignore errors, telemetry is best effort
	_ = segmentClient.Enqueue(segment.Track{
		AnonymousId: evt.AnonymousID,
		Event:       evt.eventName,
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
			Set("is_cloud", lo.Ternary(evt.CloudRegion != "", "true", "false")).
			Set("duration", evt.Duration.Milliseconds()),
		UserId: evt.UserID,
	})
	return nil
}

// UserIDFromGithubUsername hashes the github username and produces a 64-char string as userID.
// Returns an empty string if no github username is found.
func UserIDFromGithubUsername() string {
	username, err := openssh.GithubUsernameFromLocalFile()
	if err != nil || username == "" {
		return ""
	}

	const salt = "d6134cd5-347d-4b7c-a2d0-295c0f677948"
	mac := hmac.New(sha256.New, []byte(salt))

	const githubPrefix = "github:"
	mac.Write([]byte(githubPrefix + username))

	return hex.EncodeToString(mac.Sum(nil))
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
