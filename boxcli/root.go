package boxcli

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	command := &cobra.Command{
		Use: "devbox",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Don't display 'usage' on application errors.
			cmd.SilenceUsage = true
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(AddCmd())
	command.AddCommand(BuildCmd())
	command.AddCommand(GenerateCmd())
	command.AddCommand(PlanCmd())
	command.AddCommand(InitCmd())
	command.AddCommand(ShellCmd())
	return command
}

func Execute(ctx context.Context) error {
	cmd := RootCmd()
	return cmd.ExecuteContext(ctx)
}

func Main() {
	err := Execute(context.Background())
	if err != nil {
		os.Exit(1)
	}
}
