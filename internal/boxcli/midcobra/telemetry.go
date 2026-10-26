// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"os"
	"runtime/trace"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.jetify.com/devbox/internal/boxcli/featureflag"
	"go.jetify.com/devbox/internal/devbox"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/telemetry"
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

type telemetryMiddleware struct{}

// telemetryMiddleware implements interface Middleware (compile-time check)
var _ Middleware = (*telemetryMiddleware)(nil)

// skipTelemetryAnnotation is set on a command by SkipTelemetry.
const skipTelemetryAnnotation = "devbox.skip-telemetry"

// SkipTelemetry stops the telemetry middleware from recording the current run
// of cmd. Use it for commands that run so often in the background (e.g. on
// every direnv reload) that recording them isn't useful.
func SkipTelemetry(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[skipTelemetryAnnotation] = "true"
}

func telemetrySkipped(cmd *cobra.Command) bool {
	return cmd.Annotations[skipTelemetryAnnotation] != ""
}

func (m *telemetryMiddleware) preRun(cmd *cobra.Command, args []string) {
	telemetry.Start()
}

func (m *telemetryMiddleware) postRun(cmd *cobra.Command, args []string, runErr error) {
	defer trace.StartRegion(cmd.Context(), "telemetryPostRun").End()
	defer telemetry.Stop()

	if !telemetry.Enabled() {
		return
	}

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
	if telemetrySkipped(subcmd) {
		return
	}
	meta.Command = subcmd.CommandPath()
	meta.CommandFlags = flags

	meta.Packages, meta.NixpkgsHash = getPackagesAndCommitHash()
	meta.InShell = envir.IsDevboxShellEnabled()
	meta.InBrowser = envir.IsInBrowser()
	meta.InCloud = envir.IsDevboxCloud()

	if runErr != nil {
		telemetry.Error(runErr, meta)
		// TODO: This is skipping event logging of calls that end in error. We probably want to log them.
		return
	}
	telemetry.Event(telemetry.EventCommandSuccess, meta)
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

// getPackagesAndCommitHash returns the packages and nixpkgs commit of the
// project that the command opened. It reuses the already-opened project instead
// of opening it again, which would re-parse the config and reload plugins.
// Commands that don't open a project report no packages.
func getPackagesAndCommitHash() ([]string, string) {
	box := devbox.LastOpened()
	if box == nil {
		return []string{}, ""
	}
	return box.AllPackageNamesIncludingRemovedTriggerPackages(),
		box.Lockfile().Stdenv().Rev
}
