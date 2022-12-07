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

	// TODO use in the next PR instead of sshExecutablePath
	//devboxExecutablePath, err := os.Executable()
	//if err != nil {
	//	return errors.WithStack(err)
	//}
	sshExecutablePath, err := exec.LookPath("ssh")
	if err != nil {
		return errors.WithStack(err)
	}

	if err := os.Symlink(sshExecutablePath, filepath.Join(shimDir, "ssh")); err != nil {
		if !os.IsExist(err) {
			return errors.WithStack(err)
		}
	}

	scpExecutablePath, err := exec.LookPath("scp")
	if err != nil {
		return errors.WithStack(err)
	}
	if err := os.Symlink(scpExecutablePath, filepath.Join(shimDir, "scp")); err != nil {
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
