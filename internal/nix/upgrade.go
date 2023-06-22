package nix

import (
	"fmt"
	"os"
	"os/exec"

	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/internal/ux"
)

func ProfileUpgrade(ProfileDir string, pkg *Input, lock *lock.File) error {
	idx, err := ProfileListIndex(&ProfileListIndexArgs{
		Lockfile:   lock,
		Writer:     os.Stderr,
		Input:      pkg,
		ProfileDir: ProfileDir,
	})
	if err != nil {
		return err
	}
	cmd := exec.Command("nix", "profile", "upgrade",
		"--profile", ProfileDir,
		fmt.Sprintf("%d", idx),
	)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return redact.Errorf(
			"error running \"nix profile upgrade\": %s: %w", out, err)
	}
	return nil
}

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
