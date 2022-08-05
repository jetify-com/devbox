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
		RunE: runInitCmd,
	}
	return command
}

func runInitCmd(cmd *cobra.Command, args []string) error {
	path := pathArg(args)

	_, err := devbox.Init(path)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
