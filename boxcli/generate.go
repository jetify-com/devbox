package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/axiom/opensource/devbox"
)

// TODO: this command is useful for debugging.
// Decide whether we want to keep it for real – or it should be removed.
func GenerateCmd() *cobra.Command {
	command := &cobra.Command{
		Use:  "generate [<dir>]",
		Args: cobra.MaximumNArgs(1),
		RunE: runGenerateCmd,
	}
	return command
}

func runGenerateCmd(cmd *cobra.Command, args []string) error {
	path := pathArg(args)

	// Check the directory exists.
	box, err := devbox.Open(path)
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Generate()
}
