package boxcli

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/impl/devopt"
)

type hookFlags struct {
	config configFlags
}

func hookCmd() *cobra.Command {
	flags := hookFlags{}
	cmd := &cobra.Command{
		Use:   "hook [shell]",
		Short: "Print shell command to setup the shell hook to ensure an up-to-date environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := hookFunc(cmd, args, flags)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), output)
			return nil
		},
	}

	flags.config.register(cmd)
	return cmd
}

func hookFunc(cmd *cobra.Command, args []string, flags hookFlags) (string, error) {
	box, err := devbox.Open(&devopt.Opts{Dir: flags.config.path, Writer: cmd.ErrOrStderr()})
	if err != nil {
		return "", err
	}
	return box.ExportHook(args[0])
}
