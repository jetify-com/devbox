// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"fmt"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	segment "github.com/segmentio/analytics-go"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/telemetry"
)

// We collect some light telemetry to be able to improve devbox over time.
// We're aware how important privacy is and value it ourselves, so we have
// the following rules:
// 1. We only collect anonymized data – nothing that is personally identifiable
// 2. Data is only stored in SOC 2 compliant systems, and we are SOC 2 compliant ourselves.
// 3. Users should always have the ability to opt-out.
func Telemetry() Middleware {

	opts := telemetry.InitOpts()

	return &telemetryMiddleware{
		opts:     *opts,
		disabled: telemetry.IsDisabled(opts),
	}
}

type telemetryMiddleware struct {
	// Setup:
	opts     telemetry.Opts
	disabled bool

	// Used during execution:
	startTime time.Time

	executionID string
}

// telemetryMiddleware implements interface Middleware (compile-time check)
var _ Middleware = (*telemetryMiddleware)(nil)

func (m *telemetryMiddleware) preRun(cmd *cobra.Command, args []string) {
	m.startTime = telemetry.CommandStartTime()
	if !m.disabled {
		sentry := telemetry.NewSentry(m.opts.SentryDSN)
		sentry.Init(m.opts.AppName, m.opts.AppVersion, m.executionID)
	}
}

func (m *telemetryMiddleware) postRun(cmd *cobra.Command, args []string, runErr error) {
	if m.disabled {
		return
	}

	evt := m.newEventIfValid(cmd, args, runErr)
	if evt == nil {
		return
	}

	m.trackError(evt) // Sentry

	m.trackEvent(evt) // Segment
}

func (m *telemetryMiddleware) withExecutionID(execID string) Middleware {
	m.executionID = execID
	return m
}

func getSubcommand(c *cobra.Command, args []string) (subcmd *cobra.Command, subargs []string, err error) {
	if c.TraverseChildren {
		subcmd, subargs, err = c.Traverse(args)
	} else {
		subcmd, subargs, err = c.Find(args)
	}
	return subcmd, subargs, err
}

func getPackages(c *cobra.Command) []string {
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

func (m *telemetryMiddleware) trackError(evt *event) {
	if evt == nil || evt.CommandError == nil {
		// Don't send anything to sentry if the error is nil.
		return
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("command", evt.Command)
		scope.SetContext("command", map[string]interface{}{
			"subcommand": evt.Command,
			"args":       evt.CommandArgs,
			"packages":   evt.Packages,
		})
	})
	sentry.CaptureException(evt.CommandError)
}

// Consider renaming this to commandEvent
// since it has info about the specific command run.
type event struct {
	telemetry.Event
	Command       string
	CommandArgs   []string
	CommandError  error
	Failed        bool
	Packages      []string
	SentryEventID string
	Shell         string
	UserID        string
}

// newEventIfValid creates a new telemetry event, but returns nil if we cannot construct
// a valid event.
func (m *telemetryMiddleware) newEventIfValid(cmd *cobra.Command, args []string, runErr error) *event {

	subcmd, subargs, parseErr := getSubcommand(cmd, args)
	if parseErr != nil {
		// Ignore invalid commands
		return nil
	}

	pkgs := getPackages(cmd)

	// an empty userID means that we do not have a github username saved
	userID := telemetry.UserIDFromGithubUsername()

	return &event{
		Event: telemetry.Event{
			AnonymousID: telemetry.DeviceID(),
			AppName:     m.opts.AppName,
			AppVersion:  m.opts.AppVersion,
			CloudRegion: os.Getenv("DEVBOX_REGION"),
			Duration:    time.Since(m.startTime),
			OsName:      telemetry.OS(),
			UserID:      userID,
		},
		Command:      subcmd.CommandPath(),
		CommandArgs:  subargs,
		CommandError: runErr,
		Failed:       runErr != nil,
		Packages:     pkgs,
		Shell:        os.Getenv("SHELL"),
	}
}

func (m *telemetryMiddleware) trackEvent(evt *event) {
	if evt == nil {
		return
	}

	if evt.CommandError != nil {
		// verified with manual testing that the sentryID returned by CaptureException
		// is the same as m.ExecutionID, since we set EventID = m.ExecutionID in sentry.Init
		evt.SentryEventID = m.executionID
	}

	segmentClient := telemetry.NewSegmentClient(m.opts.TelemetryKey)
	defer func() {
		_ = segmentClient.Close()
	}()

	// deliberately ignore error
	_ = segmentClient.Enqueue(segment.Identify{
		AnonymousId: evt.AnonymousID,
		UserId:      evt.UserID,
	})

	_ = segmentClient.Enqueue(segment.Track{ // Ignore errors, telemetry is best effort
		AnonymousId: evt.AnonymousID, // Use device id instead
		Event:       fmt.Sprintf("[%s] Command: %s", evt.AppName, evt.Command),
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
			Set("cloud_region", evt.CloudRegion).
			Set("command", evt.Command).
			Set("command_args", evt.CommandArgs).
			Set("failed", evt.Failed).
			Set("duration", evt.Duration.Milliseconds()).
			Set("packages", evt.Packages).
			Set("sentry_event_id", evt.SentryEventID).
			Set("shell", evt.Shell),
		UserId: evt.UserID,
	})
}
