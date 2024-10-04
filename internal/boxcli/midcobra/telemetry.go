// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"os"
	"runtime/trace"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
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

type telemetryMiddleware struct{}

// telemetryMiddleware implements interface Middleware (compile-time check)
var _ Middleware = (*telemetryMiddleware)(nil)

func (m *telemetryMiddleware) preRun(cmd *cobra.Command, args []string) {
	telemetry.Start()
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

	box, err := devbox.Open(&devopt.Opts{
		Dir:            path,
		Stderr:         os.Stderr,
		IgnoreWarnings: true,
	})
	if err != nil {
		return []string{}, ""
	}

	return box.AllPackageNamesIncludingRemovedTriggerPackages(),
		box.Config().NixPkgsCommitHash()
}
