// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"context"
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
	command.AddCommand(AddCmd())
	command.AddCommand(CloudCmd())
	command.AddCommand(GenerateCmd())
	command.AddCommand(globalCmd())
	command.AddCommand(InfoCmd())
	command.AddCommand(InitCmd())
	command.AddCommand(InstallCmd())
	command.AddCommand(LogCmd())
	command.AddCommand(PlanCmd())
	command.AddCommand(RemoveCmd())
	command.AddCommand(RunCmd())
	command.AddCommand(ServicesCmd())
	command.AddCommand(SetupCmd())
	command.AddCommand(ShellCmd())
	command.AddCommand(shellEnvCmd())
	command.AddCommand(VersionCmd())
	command.AddCommand(genDocsCmd())

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
