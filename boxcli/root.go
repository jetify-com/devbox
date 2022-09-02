// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/boxcli/midcobra"
	"go.jetpack.io/devbox/build"
)

var debugMiddleware *midcobra.DebugMiddleware = &midcobra.DebugMiddleware{}

func RootCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "devbox",
		Short: "Instant, easy, predictable shells and containers",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Don't display 'usage' on application errors.
			cmd.SilenceUsage = true
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			_, err := exec.LookPath("nix-shell")
			if err != nil {
				return errors.New("could not find nix in your PATH\nInstall nix by following the instructions at https://nixos.org/download.html and make sure you've set up your PATH correctly")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(AddCmd())
	command.AddCommand(BuildCmd())
	command.AddCommand(GenerateCmd())
	command.AddCommand(InitCmd())
	command.AddCommand(PlanCmd())
	command.AddCommand(RemoveCmd())
	command.AddCommand(ShellCmd())
	command.AddCommand(VersionCmd())

	debugMiddleware.AttachToFlag(command.PersistentFlags(), "debug")

	return command
}

func Execute(ctx context.Context, args []string) int {
	defer func() {
		if r := recover(); r != nil {
			if debugMiddleware.Debug() {
				fmt.Printf("PANIC (DEBUG MODE ON): %+v\n", r)
			} else {
				fmt.Printf("Error: %s\n", r)
			}
		}
	}()
	exe := midcobra.New(RootCmd())
	exe.AddMiddleware(midcobra.Telemetry(&midcobra.TelemetryOpts{
		AppName:      "devbox",
		AppVersion:   build.Version,
		TelemetryKey: build.TelemetryKey,
	}))
	exe.AddMiddleware(debugMiddleware)
	return exe.Execute(ctx, args)
}

func Main() {
	code := Execute(context.Background(), os.Args[1:])
	os.Exit(code)
}

type runFunc func(cmd *cobra.Command, args []string) error
