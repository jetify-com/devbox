package nix

import (
	"bytes"
	"os/exec"

	"github.com/pkg/errors"
)

// ProfileInstall calls nix profile install with default profile
func ProfileInstall(nixpkgsCommit, pkg string) error {
	cmd := exec.Command("nix", "profile", "install",
		"nixpkgs/"+nixpkgsCommit+"#"+pkg,
		"--extra-experimental-features", "nix-command flakes",
	)
	cmd.Env = DefaultEnv()
	out, err := cmd.CombinedOutput()
	if bytes.Contains(out, []byte("does not provide attribute")) {
		return ErrPackageNotFound
	}

	return errors.WithStack(err)
}

func ProfileRemove(nixpkgsCommit, pkg string) error {
	info, found := flakesPkgInfo(nixpkgsCommit, pkg)
	if !found {
		return ErrPackageNotFound
	}
	cmd := exec.Command("nix", "profile", "remove",
		info.attributeKey,
		"--extra-experimental-features", "nix-command flakes",
	)
	cmd.Env = DefaultEnv()
	out, err := cmd.CombinedOutput()
	if bytes.Contains(out, []byte("does not match any packages")) {
		return ErrPackageNotInstalled
	}

	return errors.WithStack(err)
}
