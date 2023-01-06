package impl

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

func (d *Devbox) installNixProfileFlakeCommand(profileDir string) *exec.Cmd {
	cmd := exec.Command(
		"nix", "profile", "install",
		"--profile", profileDir,
		"--extra-experimental-features", "nix-command flakes",
		filepath.Join(d.configDir, ".devbox/gen/flake/"), // installables
	)
	if d.hasDevboxLockFile() {
		_ = d.copyDevboxLockToFlakeLock()
		cmd.Args = append(cmd.Args, "--no-write-lock-file")
	} else {
		cmd.Args = append(cmd.Args, "--recreate-lock-file")
	}
	return cmd
}

func (d *Devbox) copyFlakeLockToDevboxLock() error {

	flakeLock, err := os.Open(d.flakeLockPath())
	if err != nil {
		return errors.WithStack(err)
	}
	devboxLock, err := os.Create(d.devboxLockPath())
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = io.Copy(devboxLock, flakeLock)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (d *Devbox) copyDevboxLockToFlakeLock() error {
	if !d.hasDevboxLockFile() {
		return errors.New("devbox.lock file does not exist")
	}

	devboxLock, err := os.Open(d.devboxLockPath())
	if err != nil {
		return errors.WithStack(err)
	}
	flakeLock, err := os.Create(d.flakeLockPath())
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = io.Copy(flakeLock, devboxLock)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (d *Devbox) hasDevboxLockFile() bool {
	if _, err := os.Stat(d.devboxLockPath()); err != nil {
		return false
	}
	return true
}

func (d *Devbox) devboxLockPath() string {
	return filepath.Join(d.configDir, "devbox.lock")
}

func (d *Devbox) flakeLockPath() string {
	return filepath.Join(d.configDir, ".devbox/gen/flake/flake.lock")
}
