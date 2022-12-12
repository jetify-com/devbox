package sshshim

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/boxcli/featureflag"
	"go.jetpack.io/devbox/cloud/openssh"
)

const (
	configShimDir = ".config/devbox/ssh/shims"
)

// Setup creates the ssh and scp symlinks
func Setup() error {
	if featureflag.SSHShim.Disabled() {
		return nil
	}
	shimDir, err := Dir()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := openssh.EnsureDirExists(shimDir, 0744, true /*chmod*/); err != nil {
		return err
	}

	// create ssh symlink
	devboxExecutablePath, err := os.Executable()
	if err != nil {
		return errors.WithStack(err)
	}
	sshSymlink := filepath.Join(shimDir, "ssh")
	if err := makeSymlink(sshSymlink, devboxExecutablePath); err != nil {
		return errors.WithStack(err)
	}

	// create scp symlink
	scpExecutablePath, err := exec.LookPath("scp")
	if err != nil {
		return errors.WithStack(err)
	}
	scpSymlink := filepath.Join(shimDir, "scp")
	if err := makeSymlink(scpSymlink, scpExecutablePath); err != nil {
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

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WithStack(err)
	}
	shimDir := filepath.Join(home, configShimDir)
	return shimDir, nil
}
