package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/axiom/opensource/devbox"
	"go.jetpack.io/axiom/opensource/devbox/docker"
)

func BuildCmd() *cobra.Command {
	flags := &docker.BuildFlags{}

	command := &cobra.Command{
		Use:  "build [<dir>]",
		Args: cobra.MaximumNArgs(1),
		RunE: buildCmdFunc(flags),
	}

	command.Flags().BoolVar(
		&flags.NoCache, "no-cache", false, "Do not use a cache")

	return command
}

type runFunc func(cmd *cobra.Command, args []string) error

func buildCmdFunc(flags *docker.BuildFlags) runFunc {
	return func(cmd *cobra.Command, args []string) error {
		path := pathArg(args)

		// Check the directory exists.
		box, err := devbox.Open(path)
		if err != nil {
			return errors.WithStack(err)
		}

		return box.Build(docker.WithFlags(flags))
	}
}
