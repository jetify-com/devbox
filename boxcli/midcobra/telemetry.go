// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/denisbrodbeck/machineid"
	segment "github.com/segmentio/analytics-go"
	"github.com/spf13/cobra"
)

// We collect some light telemetry to be able to improve devbox over time.
// We're aware how important privacy is and value it ourselves, so we have
// the following rules:
// 1. We only collect anonymized data – nothing that is personally identifiable
// 2. Data is only stored in SOC 2 compliant systems, and we are SOC 2 compliant ourselves.
// 3. Users should always have the ability to opt-out.
func Telemetry(opts *TelemetryOpts) Middleware {
	doNotTrack, err := strconv.ParseBool(os.Getenv("DO_NOT_TRACK")) // https://consoledonottrack.com/
	if err != nil {
		doNotTrack = false
	}

	return &telemetryMiddleware{
		opts:     *opts,
		disabled: doNotTrack || opts.TelemetryKey == "",
	}
}

type TelemetryOpts struct {
	AppName      string
	AppVersion   string
	TelemetryKey string
}
type telemetryMiddleware struct {
	// Setup:
	opts     TelemetryOpts
	disabled bool

	// Used during execution:
	startTime time.Time
}

// telemetryMiddleware implements interface Middleware (compile-time check)
var _ Middleware = (*telemetryMiddleware)(nil)

func (m *telemetryMiddleware) preRun(cmd *cobra.Command, args []string) {
	m.startTime = time.Now()
}

func (m *telemetryMiddleware) postRun(cmd *cobra.Command, args []string, runErr error) {
	if m.disabled {
		return
	}

	segmentClient, _ := segment.NewWithConfig(m.opts.TelemetryKey, segment.Config{Verbose: false})

	defer func() {
		_ = segmentClient.Close()
	}()

	subcmd, subargs, parseErr := getSubcommand(cmd, args)
	if parseErr != nil {
		return // Ignore invalid commands
	}

	trackEvent(segmentClient, &event{
		AppName:     m.opts.AppName,
		AppVersion:  m.opts.AppVersion,
		Command:     subcmd.CommandPath(),
		CommandArgs: subargs,
		DeviceID:    deviceID(),
		Duration:    time.Since(m.startTime),
		Failed:      runErr != nil,
	})
}

func deviceID() string {
	salt := "64ee464f-9450-4b14-8d9c-014c0012ac1a"
	hashedID, _ := machineid.ProtectedID(salt) // Ensure machine id is hashed and non-identifiable
	return hashedID
}

func getSubcommand(c *cobra.Command, args []string) (subcmd *cobra.Command, subargs []string, err error) {
	if c.TraverseChildren {
		subcmd, subargs, err = c.Traverse(args)
	} else {
		subcmd, subargs, err = c.Find(args)
	}
	return subcmd, subargs, err
}

type event struct {
	AppName     string
	AppVersion  string
	Command     string
	CommandArgs []string
	DeviceID    string
	Duration    time.Duration
	Failed      bool
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
			Set("duration", evt.Duration.Milliseconds()),
	})
}
