package boxcli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func installCmd() *cobra.Command {
	flags := runCmdFlags{}
	command := &cobra.Command{
		Use:   "install",
		Short: "Install all packages mentioned in devbox.json",
		Long: "Start a new devbox shell and installs all packages mentioned in devbox.json in current directory or" +
			"a directory specified via --config. \n\n Then exits the shell when packages are done installing.\n\n ",
		Args:    cobra.MaximumNArgs(0),
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			// the colon ':' character in standard shell means noop.
			// So essentially, this command is running devbox run noop
			err := runScriptCmd(cmd, []string{":"}, flags)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "Finished installing packages.")
			return nil
		},
	}

	flags.config.register(command)

	return command
}
