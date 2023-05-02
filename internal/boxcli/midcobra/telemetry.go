// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"fmt"
	"os"
	"runtime/trace"
	"sort"
	"strings"
	"time"

	segment "github.com/segmentio/analytics-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/telemetry"
)

// We collect some light telemetry to be able to improve devbox over time.
// We're aware how important privacy is and value it ourselves, so we have
// the following rules:
// 1. We only collect anonymized data – nothing that is personally identifiable
// 2. Data is only stored in SOC 2 compliant systems, and we are SOC 2 compliant ourselves.
// 3. Users should always have the ability to opt-out.
func Telemetry() Middleware {
	return &telemetryMiddleware{}
}

type telemetryMiddleware struct {
	// Used during execution:
	startTime time.Time
}

// telemetryMiddleware implements interface Middleware (compile-time check)
var _ Middleware = (*telemetryMiddleware)(nil)

func (m *telemetryMiddleware) preRun(cmd *cobra.Command, args []string) {
	m.startTime = telemetry.CommandStartTime()

	telemetry.Start(telemetry.AppDevbox)
	ctx := cmd.Context()
	defer trace.StartRegion(ctx, "telemetryPreRun").End()
	if !telemetry.Enabled() {
		trace.Log(ctx, "telemetry", "telemetry is disabled")
		return
	}
}

func (m *telemetryMiddleware) postRun(cmd *cobra.Command, args []string, runErr error) {
	defer trace.StartRegion(cmd.Context(), "telemetryPostRun").End()
	defer telemetry.Stop()

	meta := telemetry.Metadata{
		FeatureFlags: featureflag.All(),
		CloudRegion:  os.Getenv(envir.DevboxRegion),
		CloudCache:   os.Getenv(envir.DevboxCache),
	}

	subcmd, flags, err := getSubcommand(cmd, args)
	if err != nil {
		// Ignore invalid commands/flags.
		return
	}
	meta.Command = subcmd.CommandPath()
	meta.CommandFlags = flags
	meta.Packages, meta.NixpkgsHash = getPackagesAndCommitHash(cmd)
	meta.InShell = envir.IsDevboxShellEnabled()
	meta.InBrowser = envir.IsInBrowser()
	meta.InCloud = envir.IsDevboxCloud()
	telemetry.Error(runErr, meta)

	if !telemetry.Enabled() {
		return
	}
	evt := m.newEventIfValid(cmd, args, runErr)
	if evt == nil {
		return
	}
	m.trackEvent(evt) // Segment
}

// Consider renaming this to commandEvent
// since it has info about the specific command run.
type event struct {
	telemetry.Event
	Command       string
	CommandArgs   []string
	CommandError  error
	CommandHidden bool
	Failed        bool
	Packages      []string
	CommitHash    string // the nikpkgs commit hash in devbox.json
	InDevboxShell bool
	DevboxEnv     map[string]any // Devbox-specific environment variables
	SentryEventID string
	Shell         string
}

// newEventIfValid creates a new telemetry event, but returns nil if we cannot construct
// a valid event.
func (m *telemetryMiddleware) newEventIfValid(cmd *cobra.Command, args []string, runErr error) *event {
	subcmd, flags, parseErr := getSubcommand(cmd, args)
	if parseErr != nil {
		// Ignore invalid commands
		return nil
	}

	pkgs, hash := getPackagesAndCommitHash(cmd)

	// an empty userID means that we do not have a github username saved
	userID := telemetry.UserIDFromGithubUsername()

	devboxEnv := map[string]interface{}{}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "DEVBOX") && strings.Contains(e, "=") {
			key := strings.Split(e, "=")[0]
			devboxEnv[key] = os.Getenv(key)
		}
	}

	return &event{
		Event: telemetry.Event{
			AnonymousID: telemetry.DeviceID,
			AppName:     telemetry.AppDevbox,
			AppVersion:  build.Version,
			CloudRegion: os.Getenv(envir.DevboxRegion),
			Duration:    time.Since(m.startTime),
			OsName:      build.OS(),
			UserID:      userID,
		},
		Command:      subcmd.CommandPath(),
		CommandArgs:  flags,
		CommandError: runErr,
		// The command is hidden if either the top-level command is hidden or
		// the specific sub-command that was executed is hidden.
		CommandHidden: cmd.Hidden || subcmd.Hidden,
		Failed:        runErr != nil,
		Packages:      pkgs,
		CommitHash:    hash,
		InDevboxShell: envir.IsDevboxShellEnabled(),
		DevboxEnv:     devboxEnv,
		Shell:         os.Getenv(envir.Shell),
	}
}

func (m *telemetryMiddleware) trackEvent(evt *event) {
	if evt == nil || evt.CommandHidden {
		return
	}

	if evt.CommandError != nil {
		evt.SentryEventID = telemetry.ExecutionID
	}
	segmentClient := telemetry.NewSegmentClient(build.TelemetryKey)
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

func getSubcommand(cmd *cobra.Command, args []string) (subcmd *cobra.Command, flags []string, err error) {
	if cmd.TraverseChildren {
		subcmd, _, err = cmd.Traverse(args)
	} else {
		subcmd, _, err = cmd.Find(args)
	}

	subcmd.Flags().Visit(func(f *pflag.Flag) {
		flags = append(flags, "--"+f.Name)
	})
	sort.Strings(flags)
	return subcmd, flags, err
}

func getPackagesAndCommitHash(c *cobra.Command) ([]string, string) {
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
		return []string{}, ""
	}

	return box.Config().Packages, box.Config().Nixpkgs.Commit
}
