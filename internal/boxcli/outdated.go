package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
)

func outdatedCmd() *cobra.Command {
	flags := listCmdFlags{}
	command := &cobra.Command{
		Use:     "outdated",
		Short:   "Show all outdated packages",
		Args:    cobra.MaximumNArgs(0),
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, _ []string) error {
			box, err := devbox.Open(&devopt.Opts{
				Dir:    flags.config.path,
				Stderr: cmd.ErrOrStderr(),
			})
			if err != nil {
				return errors.WithStack(err)
			}

			resutls, err := box.Outdated(cmd.Context())
			if err != nil {
				return errors.WithStack(err)
			}

			if len(resutls) == 0 {
				cmd.Println("Your packages are up to date!")
				return nil
			}

			cmd.Println("The following packages can be updated:")
			for pkg, version := range resutls {
				cmd.Printf(" * %-30s %s -> %s\n", pkg, version.Current, version.Latest)
			}
			return nil
		},
	}

	return command
}
