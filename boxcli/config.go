package boxcli

import (
	"github.com/spf13/cobra"
)

// to be composed into xyzCmdFlags structs
type configFlags struct {
	path string
}

func (flags *configFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&flags.path, "config", "c", "", "path to directory containing a devbox.json config file",
	)
}
