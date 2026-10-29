// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/boxcli/midcobra"
	"go.jetify.com/devbox/internal/cmdutil"
	"go.jetify.com/devbox/internal/debug"
	"go.jetify.com/devbox/internal/launcher"
	"go.jetify.com/devbox/internal/telemetry"
	"go.jetify.com/devbox/internal/vercheck"
)

var (
	debugMiddleware = &midcobra.DebugMiddleware{}
	traceMiddleware = &midcobra.TraceMiddleware{}
)

type rootCmdFlags struct {
	quiet bool
}

func RootCmd() *cobra.Command {
	// Run the persistent hooks of every ancestor command instead of only the
	// closest one, so subcommands that declare their own PersistentPreRunE
	// don't skip the root's.
	cobra.EnableTraverseRunHooks = true

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
	command.AddCommand(createCmd())
	command.AddCommand(generateCmd())
	command.AddCommand(globalCmd())
	command.AddCommand(infoCmd())
	command.AddCommand(initCmd())
	command.AddCommand(installCmd())
	command.AddCommand(integrateCmd())
	command.AddCommand(listCmd())
	command.AddCommand(logCmd())
	command.AddCommand(patchCmd())
	command.AddCommand(removeCmd())
	command.AddCommand(runCmd(runFlagDefaults{}))
	command.AddCommand(searchCmd())
	command.AddCommand(servicesCmd())
	command.AddCommand(setupCmd())
	command.AddCommand(shellCmd(shellFlagDefaults{}))
	command.AddCommand(shellEnvCmd(shellenvFlagDefaults{
		recomputeEnv: true,
	}))
	command.AddCommand(updateCmd())
	command.AddCommand(versionCmd())
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
		&flags.quiet, "quiet", "q", false, "suppresses logs",
	)
	debugMiddleware.AttachToFlag(command.PersistentFlags(), "debug")
	traceMiddleware.AttachToFlag(command.PersistentFlags(), "trace")

	return command
}

func Execute(ctx context.Context, args []string) int {
	defer debug.Recover()
	rootCmd := RootCmd()
	exe := midcobra.New(rootCmd)
	// Runs first because it might re-run the command with another devbox
	// version, which makes the work done by any earlier middleware wasted.
	exe.AddMiddleware(midcobra.ProjectVersion())
	exe.AddMiddleware(traceMiddleware)
	exe.AddMiddleware(midcobra.Telemetry())
	exe.AddMiddleware(debugMiddleware)
	return exe.Execute(ctx, wrapArgsForRun(rootCmd, args))
}

func Main() {
	// Runs first because it might run another devbox version instead of this
	// one, which makes any work done before it wasted.
	launcher.Main()

	timer := debug.Timer(strings.Join(os.Args, " "))
	setSystemBinaryPaths()
	ctx := context.Background()

	if len(os.Args) > 1 && os.Args[1] == "upload-telemetry" {
		// This subcommand is hidden and only run by devbox itself as a
		// child process. We need to really make sure that we always
		// exit and don't leave orphaned processes laying around.
		time.AfterFunc(5*time.Second, func() {
			os.Exit(0)
		})
		telemetry.Upload()
		return
	}

	code := Execute(ctx, os.Args[1:])
	// Run out here instead of as a middleware so we can capture any time we spend
	// in middlewares as well.
	timer.End()
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

func setSystemBinaryPaths() {
	if os.Getenv("DEVBOX_SYSTEM_BASH") == "" {
		os.Setenv("DEVBOX_SYSTEM_BASH", cmdutil.GetPathOrDefault("bash", "/bin/bash"))
	}
	if os.Getenv("DEVBOX_SYSTEM_SED") == "" {
		os.Setenv("DEVBOX_SYSTEM_SED", cmdutil.GetPathOrDefault("sed", "/usr/bin/sed"))
	}
}
