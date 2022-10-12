package boxcli

import (
	"github.com/spf13/cobra"
)

// Keep this env-var name same as its usage in shell.nix.tmpl
const shellConfigEnvVar = "DEVBOX_SHELL_CONFIG"

// to be composed into xyzCmdFlags structs
type configFlags struct {
	path string
}

func registerConfigFlags(cmd *cobra.Command, flags *configFlags) {
	cmd.Flags().StringVarP(
		&flags.path, "config", "c", "", "path to directory containing a devbox.json config file",
	)
}
