package nix

import (
	"os"
	"os/exec"

	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/internal/ux"
)

func FlakeUpdate(ProfileDir string) error {
	ux.Finfo(os.Stderr, "Running \"nix flake update\"\n")
	cmd := exec.Command("nix", "flake", "update", ProfileDir)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return redact.Errorf(
			"error running \"nix flake update\": %s: %w", out, err)

	}
	return nil
}
