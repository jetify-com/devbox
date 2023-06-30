package nixprofile

import (
	"fmt"
	"os"
	"os/exec"

	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/redact"
)

func ProfileUpgrade(ProfileDir string, pkg *nix.Package, lock *lock.File) error {
	idx, err := ProfileListIndex(
		&ProfileListIndexArgs{
			Lockfile:   lock,
			Writer:     os.Stderr,
			Input:      pkg,
			ProfileDir: ProfileDir,
		},
	)
	if err != nil {
		return err
	}
	cmd := exec.Command(
		"nix", "profile", "upgrade",
		"--profile", ProfileDir,
		fmt.Sprintf("%d", idx),
	)
	cmd.Args = append(cmd.Args, nix.ExperimentalFlags()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return redact.Errorf(
			"error running \"nix profile upgrade\": %s: %w", out, err,
		)
	}
	return nil
}
