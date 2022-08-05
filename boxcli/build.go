package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/axiom/opensource/devbox"
)

func BuildCmd() *cobra.Command {
	command := &cobra.Command{
		Use:  "build [<dir>]",
		Args: cobra.MaximumNArgs(1),
		RunE: runBuildCmd,
	}
	return command
}

func runBuildCmd(cmd *cobra.Command, args []string) error {
	path := pathArg(args)

	// Check the directory exists.
	box, err := devbox.Open(path)
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Build()
}
