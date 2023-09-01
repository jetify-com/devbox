package nix

import (
	"context"
	"os/exec"
)

func command(args ...string) *exec.Cmd {
	return commandContext(context.Background(), args...)
}

func commandContext(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "nix", args...)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	return cmd
}

func allowUnfreeEnv(curEnv []string) []string {
	return append(curEnv, "NIXPKGS_ALLOW_UNFREE=1")
}

func allowInsecureEnv(curEnv []string) []string {
	return append(curEnv, "NIXPKGS_ALLOW_INSECURE=1")
}
