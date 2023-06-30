package nix

import (
	"os"
	"os/exec"
)

func command(args ...string) *exec.Cmd {

	cmd := exec.Command("nix", args...)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	return cmd
}

func allowUnfreeEnv() []string {
	return append(os.Environ(), "NIXPKGS_ALLOW_UNFREE=1")
}
