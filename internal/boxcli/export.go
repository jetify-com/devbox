package boxcli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// exportCmd is an alias of shellenv, but is also hidden and hence we cannot define it
// simply using `Aliases: []string{"export"}` in the shellEnvCmd definition.
func exportCmd() *cobra.Command {
	flags := shellEnvCmdFlags{}
	cmd := &cobra.Command{
		Use:    "export [shell]",
		Hidden: true,
		Short:  "Print shell command to setup the shell export to ensure an up-to-date environment",
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shellEnvFunc(cmd, flags)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), s)
			return nil
		},
	}

	registerShellEnvFlags(cmd, &flags)
	return cmd
}
