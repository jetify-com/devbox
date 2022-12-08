// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/boxcli/midcobra"
	"go.jetpack.io/devbox/build"
	"go.jetpack.io/devbox/cloud/openssh/sshshim"
	"go.jetpack.io/devbox/debug"
)

var debugMiddleware *midcobra.DebugMiddleware = &midcobra.DebugMiddleware{}

func RootCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "devbox",
		Short: "Instant, easy, predictable development environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	command.AddCommand(AddCmd())
	command.AddCommand(BuildCmd())
	command.AddCommand(CloudCmd())
	command.AddCommand(GenerateCmd())
	command.AddCommand(InfoCmd())
	command.AddCommand(InitCmd())
	command.AddCommand(PlanCmd())
	command.AddCommand(RemoveCmd())
	command.AddCommand(RunCmd())
	command.AddCommand(ServicesCmd())
	command.AddCommand(SetupCmd())
	command.AddCommand(ShellCmd())
	command.AddCommand(VersionCmd())
	command.AddCommand(genDocsCmd())

	debugMiddleware.AttachToFlag(command.PersistentFlags(), "debug")

	return command
}

func Execute(ctx context.Context, args []string) int {
	defer debug.Recover()
	exe := midcobra.New(RootCmd())
	exe.AddMiddleware(midcobra.Telemetry(&midcobra.TelemetryOpts{
		AppName:      "devbox",
		AppVersion:   build.Version,
		SentryDSN:    build.SentryDSN,
		TelemetryKey: build.TelemetryKey,
	}))
	exe.AddMiddleware(debugMiddleware)
	return exe.Execute(ctx, args)
}

func executeSSH() int {
	sshshim.EnableDebug() // Always enable for now.
	debug.Log("os.Args: %v", os.Args)
	if err := sshshim.InvokeSSHCommand(); err != nil {
		debug.Log("ERROR: %v", err)
		fmt.Fprintf(os.Stderr, "%v", err)
		return 1
	}
	return 0
}

func Main() {
	if strings.HasSuffix(os.Args[0], "ssh") {
		code := executeSSH()
		os.Exit(code)
	}
	code := Execute(context.Background(), os.Args[1:])
	os.Exit(code)
}
