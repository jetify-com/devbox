package sshshim

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/boxcli/featureflag"
	"go.jetpack.io/devbox/cloud/mutagenbox"
	"go.jetpack.io/devbox/cloud/openssh"
)

// Setup creates the ssh and scp symlinks
func Setup() error {
	if featureflag.SSHShim.Disabled() {
		return nil
	}
	shimDir, err := mutagenbox.ShimDir()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := openssh.EnsureDirExists(shimDir, 0744, true /*chmod*/); err != nil {
		return err
	}

	devboxExecutablePath, err := os.Executable()
	if err != nil {
		return errors.WithStack(err)
	}

	// create ssh symlink
	sshSymlink := filepath.Join(shimDir, "ssh")
	if err := makeSymlink(sshSymlink, devboxExecutablePath); err != nil {
		return errors.WithStack(err)
	}

	// create scp symlink
	scpSymlink := filepath.Join(shimDir, "scp")
	if err := makeSymlink(scpSymlink, devboxExecutablePath); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func makeSymlink(from string, target string) error {

	if err := os.Remove(from); err != nil && !os.IsNotExist(err) {
		return errors.WithStack(err)
	}

	if err := os.Symlink(target, from); err != nil {
		if !os.IsExist(err) {
			return errors.WithStack(err)
		}
	}
	return nil
}
