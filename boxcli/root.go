package boxcli

import (
	"context"

	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	command := &cobra.Command{
		Use: "devbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(AddCmd())
	command.AddCommand(BuildCmd())
	command.AddCommand(GenerateCmd())
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
		panic(err)
	}
}
