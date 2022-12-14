package sshshim

import (
	"context"
	"fmt"
	"os"

	"go.jetpack.io/devbox/internal/boxcli/midcobra"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/telemetry"
)

func Execute(ctx context.Context, args []string) int {
	defer debug.Recover()
	sentry := initSentry()

	err := execute(args)

	sentry.CaptureException(err)

	if err != nil {
		return 1
	}
	return 0
}

func execute(args []string) error {
	EnableDebug() // Always enable for now.
	debug.Log("os.Args: %v", args)

	if alive, err := EnsureLiveVMOrTerminateMutagenSessions(args[1:]); err != nil {
		debug.Log("ensureLiveVMOrTerminateMutagenSessions error: %v", err)
		fmt.Fprintf(os.Stderr, "%v", err)
		return err
	} else if !alive {
		return nil
	}

	if err := InvokeSSHOrSCPCommand(args); err != nil {
		debug.Log("InvokeSSHorSCPCommand error: %v", err)
		fmt.Fprintf(os.Stderr, "%v", err)
		return err
	}
	return nil
}

func initSentry() *telemetry.Sentry {
	const appName = "devbox-sshshim"
	s := telemetry.NewSentry(build.SentryDSN)
	s.Init(appName, build.Version, midcobra.ExecutionID())

	return s
}
