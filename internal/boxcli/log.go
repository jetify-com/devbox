package boxcli

import (
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/telemetry"
)

func LogCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:    "log <event-name> [<event-specific-args>]",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return doLogCommand(cmd, args)
		},
	}

	return cmd
}

func doLogCommand(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return usererr.New("expect an <event-name> arg for command: %s", cmd.CommandPath())
	}

	if args[0] == "shell-ready" || args[0] == "shell-interactive" {
		if len(args) < 2 {
			return usererr.New("expected a start-time argument for logging the shell-ready event")
		}
		return telemetry.LogShellDurationEvent(args[0] /*event name*/, args[1] /*startTime*/)
	}
	return usererr.New("unrecognized event-name %s for command: %s", args[0], cmd.CommandPath())
}
