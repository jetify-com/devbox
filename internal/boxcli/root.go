// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/boxcli/midcobra"
	"go.jetpack.io/devbox/internal/cloud/openssh/sshshim"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/telemetry"
	"go.jetpack.io/devbox/internal/vercheck"
)

var (
	debugMiddleware *midcobra.DebugMiddleware = &midcobra.DebugMiddleware{}
	traceMiddleware *midcobra.TraceMiddleware = &midcobra.TraceMiddleware{}
)

type rootCmdFlags struct {
	quiet bool
}

func RootCmd() *cobra.Command {
	flags := rootCmdFlags{}
	command := &cobra.Command{
		Use:   "devbox",
		Short: "Instant, easy, predictable development environments",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			vercheck.CheckLauncherVersion(cmd.ErrOrStderr())
			if flags.quiet {
				cmd.SetErr(io.Discard)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	// Stable commands
	command.AddCommand(addCmd())
	command.AddCommand(generateCmd())
	command.AddCommand(globalCmd())
	command.AddCommand(infoCmd())
	command.AddCommand(initCmd())
	command.AddCommand(installCmd())
	command.AddCommand(logCmd())
	command.AddCommand(planCmd())
	command.AddCommand(removeCmd())
	command.AddCommand(runCmd())
	command.AddCommand(servicesCmd())
	command.AddCommand(setupCmd())
	command.AddCommand(shellCmd())
	command.AddCommand(shellEnvCmd())
	command.AddCommand(versionCmd())
	// Preview commands
	command.AddCommand(cloudCmd())
	// Internal commands
	command.AddCommand(genDocsCmd())

	// Register the "all" command to list all commands, including hidden ones.
	// This makes debugging easier.
	command.AddCommand(&cobra.Command{
		Use:    "all",
		Short:  "List all commands, including hidden ones",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			listAllCommands(command, "")
		},
	})

	command.PersistentFlags().BoolVarP(
		&flags.quiet, "quiet", "q", false, "suppresses logs")
	debugMiddleware.AttachToFlag(command.PersistentFlags(), "debug")
	traceMiddleware.AttachToFlag(command.PersistentFlags(), "trace")

	return command
}

func Execute(ctx context.Context, args []string) int {
	defer debug.Recover()
	exe := midcobra.New(RootCmd())
	exe.AddMiddleware(traceMiddleware)
	exe.AddMiddleware(midcobra.Telemetry())
	exe.AddMiddleware(debugMiddleware)
	return exe.Execute(ctx, args)
}

func Main() {
	if strings.HasSuffix(os.Args[0], "ssh") ||
		strings.HasSuffix(os.Args[0], "scp") {
		code := sshshim.Execute(os.Args)
		os.Exit(code)
	}
	if len(os.Args) > 1 && os.Args[1] == "bug" {
		telemetry.ReportErrors()
		return
	}
	code := Execute(context.Background(), os.Args[1:])
	os.Exit(code)
}

func listAllCommands(cmd *cobra.Command, indent string) {
	// Print this command's name and description in table format with indentation
	fmt.Printf("%s%-20s%s\n", indent, cmd.Use, cmd.Short)

	// Recursively list child commands with increased indentation
	for _, childCmd := range cmd.Commands() {
		listAllCommands(childCmd, indent+"\t")
	}
}
