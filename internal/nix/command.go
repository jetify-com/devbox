package nix

import (
	"os/exec"
)

func command(args ...string) *exec.Cmd {

	cmd := exec.Command("nix", args...)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	return cmd
}

func allowUnfreeEnv(curEnv []string) []string {
	return append(curEnv, "NIXPKGS_ALLOW_UNFREE=1")
}

func allowInsecureEnv(curEnv []string) []string {
	return append(curEnv, "NIXPKGS_ALLOW_INSECURE=1")
}
