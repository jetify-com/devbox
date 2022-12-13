// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/getsentry/sentry-go"
	segment "github.com/segmentio/analytics-go"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

// We collect some light telemetry to be able to improve devbox over time.
// We're aware how important privacy is and value it ourselves, so we have
// the following rules:
// 1. We only collect anonymized data – nothing that is personally identifiable
// 2. Data is only stored in SOC 2 compliant systems, and we are SOC 2 compliant ourselves.
// 3. Users should always have the ability to opt-out.
func Segment(opts *SegmentOpts) Middleware {
	return &segmentMiddleware{
		opts:     *opts,
		disabled: doNotTrack() || opts.TelemetryKey == "",
	}
}

func doNotTrack() bool {
	// https://consoledonottrack.com/
	doNotTrack_, err := strconv.ParseBool(os.Getenv("DO_NOT_TRACK"))
	if err != nil {
		doNotTrack_ = false
	}
	return doNotTrack_
}

type SegmentOpts struct {
	AppName      string
	AppVersion   string
	TelemetryKey string
}
type segmentMiddleware struct {
	// Setup:
	opts     SegmentOpts
	disabled bool

	// Used during execution:
	startTime time.Time

	executionID string
}

// segmentMiddleware implements interface Middleware (compile-time check)
var _ Middleware = (*segmentMiddleware)(nil)

func (m *segmentMiddleware) preRun(cmd Command, args []string) {
	m.startTime = time.Now()
}

func (m *segmentMiddleware) postRun(cmd Command, args []string, runErr error) {
	if m.disabled {
		return
	}
	segmentClient, _ := segment.NewWithConfig(m.opts.TelemetryKey, segment.Config{
		BatchSize: 1, /* no batching */
		// Discard logs:
		Logger:  segment.StdLogger(log.New(io.Discard, "" /* prefix */, 0)),
		Verbose: false,
	})

	defer func() {
		_ = segmentClient.Close()
	}()

	subcmd, subargs, parseErr := getSubcommand(cmd, args)
	if parseErr != nil {
		return // Ignore invalid commands
	}

	var sentryEventID string
	if runErr != nil {
		defer sentry.Flush(2 * time.Second)
		_ /*eventIDPointer*/ = sentry.CaptureException(runErr)
		sentryEventID = m.executionID
		// verified with manual testing that the sentryID returned by CaptureException
		// is the same as m.executionID, since we set EventID = m.executionID in initSentry()
	}

	trackEvent(segmentClient, &event{
		AppName:       m.opts.AppName,
		AppVersion:    m.opts.AppVersion,
		Command:       subcmd.CommandPath(),
		CommandArgs:   subargs,
		DeviceID:      deviceID(),
		Duration:      time.Since(m.startTime),
		Failed:        runErr != nil,
		Packages:      getPackages(cmd),
		SentryEventID: sentryEventID,
	})
}

func (m *segmentMiddleware) withExecutionID(execID string) Middleware {
	m.executionID = execID
	return m
}

func deviceID() string {
	salt := "64ee464f-9450-4b14-8d9c-014c0012ac1a"
	hashedID, _ := machineid.ProtectedID(salt) // Ensure machine id is hashed and non-identifiable
	return hashedID
}

func getSubcommand(c Command, args []string) (subcmd *cobra.Command, subargs []string, err error) {
	if c.ShouldTraverseChildren() {
		subcmd, subargs, err = c.Traverse(args)
	} else {
		subcmd, subargs, err = c.Find(args)
	}
	return subcmd, subargs, err
}

func getPackages(c Command) []string {
	configFlag := c.Flag("config")
	// for shell, run, and add command, path can be set via --config
	// if --config is not set, default to current directory which is ""
	// the only exception is the init command, for the path can be set with args
	// since after running init there will be no packages set in devbox.json
	// we can safely ignore this case.
	var path string
	if configFlag != nil {
		path = configFlag.Value.String()
	}

	box, err := devbox.Open(path, os.Stdout)
	if err != nil {
		return []string{}
	}

	return box.Config().Packages
}

type event struct {
	AppName       string
	AppVersion    string
	Command       string
	CommandArgs   []string
	DeviceID      string
	Duration      time.Duration
	Failed        bool
	Packages      []string
	SentryEventID string
}

func trackEvent(client segment.Client, evt *event) {
	_ = client.Enqueue(segment.Track{ // Ignore errors, telemetry is best effort
		AnonymousId: evt.DeviceID, // Use device id instead
		Event:       fmt.Sprintf("[%s] Command: %s", evt.AppName, evt.Command),
		Context: &segment.Context{
			Device: segment.DeviceInfo{
				Id: evt.DeviceID,
			},
			App: segment.AppInfo{
				Name:    evt.AppName,
				Version: evt.AppVersion,
			},
			OS: segment.OSInfo{
				Name: runtime.GOOS,
			},
		},
		Properties: segment.NewProperties().
			Set("command", evt.Command).
			Set("command_args", evt.CommandArgs).
			Set("failed", evt.Failed).
			Set("duration", evt.Duration.Milliseconds()).
			Set("packages", evt.Packages).
			Set("sentry_event_id", evt.SentryEventID),
	})
}
