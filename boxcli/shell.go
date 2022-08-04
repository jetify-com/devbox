package boxcli

import (
	"github.com/spf13/cobra"
	"go.jetpack.io/axiom/opensource/devbox"
)

func ShellCmd() *cobra.Command {
	command := &cobra.Command{
		Use:  "shell",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := args[0]
			devbox.Shell(path)
		},
	}
	return command
}
