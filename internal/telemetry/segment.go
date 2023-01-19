package telemetry

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	segment "github.com/segmentio/analytics-go"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/cloud/openssh"
)

func NewSegmentClient(telemetryKey string) segment.Client {
	segmentClient, _ := segment.NewWithConfig(telemetryKey, segment.Config{
		BatchSize: 1, /* no batching */
		// Discard logs:
		Logger:  segment.StdLogger(log.New(io.Discard, "" /* prefix */, 0)),
		Verbose: false,
	})

	return segmentClient
}

var cmdStartTime time.Time

func CommandStartTime() time.Time {
	if cmdStartTime.IsZero() {
		cmdStartTime = time.Now()
	}
	return cmdStartTime
}

type Event struct {
	AnonymousID string
	AppName     string
	AppVersion  string
	CloudRegion string
	OsName      string
	UserID      string
}

type ttiEvent struct {
	Event
	durationSeconds int
}

func LogShellTimeToInteractiveEvent(startTime string) error {
	start, err := timeFromUnixTimestamp(startTime)
	if err != nil {
		return errors.WithStack(err)
	}

	evt := ttiEvent{
		Event: Event{
			AnonymousID: DeviceID(),
			AppName:     "",
			AppVersion:  "",
			CloudRegion: os.Getenv("DEVBOX_REGION"),
			OsName:      OS(),
			UserID:      UserIDFromGithubUsername(),
		},
		durationSeconds: int(math.Round(time.Since(start).Seconds())),
	}

	return logShellTimeToInteractiveEvent(evt)
}

func logShellTimeToInteractiveEvent(evt ttiEvent) error {
	fmt.Printf("DEBUG: logging with duration %d\n", evt.durationSeconds)
	if build.TelemetryKey == "" {
		// disabled
		return nil
	}

	segmentClient := NewSegmentClient(build.TelemetryKey)
	defer func() {
		_ = segmentClient.Close()
	}()

	// Ignore errors, telemetry is best effort
	_ = segmentClient.Enqueue(segment.Track{
		AnonymousId: evt.AnonymousID,
		Event:       "shell-time-to-interactive",
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
			Set("duration_seconds", evt.durationSeconds),
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

func timeFromUnixTimestamp(timestamp string) (time.Time, error) {

	i, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Time{}, errors.WithStack(err)
	}
	return time.Unix(i, 0), nil
}
