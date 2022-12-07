package sshshim

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/boxcli/featureflag"
	"go.jetpack.io/devbox/cloud/openssh"
)

const (
	configShimDir = ".config/devbox/shims"
)

const sshShimContents = `#!/bin/sh
# Next PR:
# devbox cloud ssh "$@"
ssh "$@"
`

const scpShimContents = `#!/bin/sh
scp "$@"
`

// Setup creates the ssh and scp files
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

	if err := os.WriteFile(filepath.Join(shimDir, "ssh"), []byte(sshShimContents), 0744); err != nil {
		return errors.WithStack(err)
	}

	if err := os.WriteFile(filepath.Join(shimDir, "scp"), []byte(scpShimContents), 0744); err != nil {
		return errors.WithStack(err)
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
