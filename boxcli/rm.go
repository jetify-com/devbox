package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/axiom/opensource/devbox"
)

func RemoveCmd() *cobra.Command {
	command := &cobra.Command{
		Use:  "rm <pkg>...",
		Args: cobra.MinimumNArgs(1),
		RunE: runRemoveCmd,
	}
	return command
}

func runRemoveCmd(cmd *cobra.Command, args []string) error {
	box, err := devbox.Open(".")
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Remove(args...)
}
