package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/axiom/opensource/devbox"
)

func AddCmd() *cobra.Command {
	command := &cobra.Command{
		Use:  "add <pkg>...",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			box, err := devbox.Open(".")
			if err != nil {
				return errors.WithStack(err)
			}

			return box.Add(args...)
		},
	}
	return command
}
