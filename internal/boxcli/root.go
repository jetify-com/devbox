// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/midcobra"
	"go.jetpack.io/devbox/internal/cloud/openssh/sshshim"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/telemetry"
	"go.jetpack.io/devbox/internal/vercheck"
)

var (
	debugMiddleware = &midcobra.DebugMiddleware{}
	traceMiddleware = &midcobra.TraceMiddleware{}
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
			if flags.quiet {
				cmd.SetErr(io.Discard)
			}
			vercheck.CheckVersion(cmd.ErrOrStderr(), cmd.CommandPath())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	// Stable commands
	command.AddCommand(addCmd())
	if featureflag.Auth.Enabled() {
		command.AddCommand(authCmd())
	}
	command.AddCommand(createCmd())
	command.AddCommand(generateCmd())
	command.AddCommand(globalCmd())
	command.AddCommand(infoCmd())
	command.AddCommand(initCmd())
	command.AddCommand(installCmd())
	command.AddCommand(integrateCmd())
	command.AddCommand(logCmd())
	command.AddCommand(planCmd())
	command.AddCommand(removeCmd())
	command.AddCommand(runCmd())
	command.AddCommand(searchCmd())
	command.AddCommand(servicesCmd())
	command.AddCommand(setupCmd())
	command.AddCommand(shellCmd())
	command.AddCommand(shellEnvCmd())
	command.AddCommand(updateCmd())
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
	ctx := context.Background()
	if strings.HasSuffix(os.Args[0], "ssh") ||
		strings.HasSuffix(os.Args[0], "scp") {
		os.Exit(sshshim.Execute(ctx, os.Args))
	}

	if len(os.Args) > 1 && os.Args[1] == "bug" {
		telemetry.ReportErrors()
		return
	}

	os.Exit(Execute(ctx, os.Args[1:]))
}

func listAllCommands(cmd *cobra.Command, indent string) {
	// Print this command's name and description in table format with indentation
	fmt.Printf("%s%-20s%s\n", indent, cmd.Use, cmd.Short)

	// Recursively list child commands with increased indentation
	for _, childCmd := range cmd.Commands() {
		listAllCommands(childCmd, indent+"\t")
	}
}
