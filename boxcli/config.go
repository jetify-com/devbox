package boxcli

import (
	"github.com/spf13/cobra"
)

const currentDir = "."

// to be composed into xyzCmdFlags structs
type configFlags struct {
	path string
}

func registerConfigFlags(cmd *cobra.Command, flags *configFlags) {
	cmd.Flags().StringVarP(
		&flags.path, "config", "c", currentDir, "path to directory containing a devbox.json config file",
	)
}
