package boxcli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	command := &cobra.Command{
		Use: "devbox",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Ran devbox")
		},
	}
	command.AddCommand(ShellCmd())
	return command
}

func Execute(ctx context.Context) {
	cmd := RootCmd()
	_ = cmd.ExecuteContext(ctx)
}

func Main() {
	Execute(context.Background())
}
