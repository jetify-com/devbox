package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/axiom/opensource/devbox"
)

func InitCmd() *cobra.Command {
	command := &cobra.Command{
		Use:  "init [<dir>]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to the current working directory
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			// Check the directory exists.
			box, err := devbox.Open(path)
			if err != nil {
				return errors.WithStack(err)
			}

			return box.Init()
		},
	}
	return command
}
