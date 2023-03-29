package sshshim

import (
	"fmt"
	"os"

	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/telemetry"
)

func Execute(args []string) int {
	defer debug.Recover()
	telemetry.Start(telemetry.AppSSHShim)
	defer telemetry.Stop()

	if err := execute(args); err != nil {
		telemetry.Error(err, telemetry.Metadata{})
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
