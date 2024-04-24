// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package sshshim

import (
	"context"
	"fmt"
	"os"

	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/telemetry"
)

func Execute(ctx context.Context, args []string) int {
	defer debug.Recover()
	telemetry.Start()
	defer telemetry.Stop()

	if err := execute(ctx, args); err != nil {
		telemetry.Error(err, telemetry.Metadata{})
		return 1
	}
	return 0
}

func execute(ctx context.Context, args []string) error {
	EnableDebug() // Always enable for now.
	debug.Log("os.Args: %v", args)

	alive, err := EnsureLiveVMOrTerminateMutagenSessions(ctx, args[1:])
	if err != nil {
		debug.Log("ensureLiveVMOrTerminateMutagenSessions error: %v", err)
		fmt.Fprintf(os.Stderr, "%v", err)
		return err
	}
	if !alive {
		return nil
	}

	if err := InvokeSSHOrSCPCommand(args); err != nil {
		debug.Log("InvokeSSHorSCPCommand error: %v", err)
		fmt.Fprintf(os.Stderr, "%v", err)
		return err
	}

	return nil
}
