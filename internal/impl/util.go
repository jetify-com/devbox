// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/integrations/envsec"
	"go.jetpack.io/devbox/internal/nix/nixprofile"

	"go.jetpack.io/devbox/internal/xdg"
)

// addDevboxUtilityPackage adds a package to the devbox utility profile.
// It's used to install applications devbox might need, like process-compose
// This is an alternative to a global install which would modify a user's
// environment.
func (d *Devbox) addDevboxUtilityPackage(ctx context.Context, pkg string) error {
	profilePath, err := utilityNixProfilePath()
	if err != nil {
		return err
	}

	return nixprofile.ProfileInstall(ctx, &nixprofile.ProfileInstallArgs{
		Lockfile:    d.lockfile,
		Package:     pkg,
		ProfilePath: profilePath,
		Writer:      d.stderr,
	})
}

// addDevboxUtilityPackages adds binaries that we want the user to have access
// to (e.g. envsec).
// Question: Should we add utilityBinPath here? That would allow user to use
// process-compose, etc
func (d *Devbox) addUtilitiesToPath(path string) (string, error) {
	if d.cfg.IsEnvsecEnabled() {
		envsecPath, err := envsec.EnsureInstalled()
		if err != nil {
			return "", err
		}
		path = path + string(os.PathListSeparator) + filepath.Dir(envsecPath)
	}
	return path, nil
}

func utilityLookPath(binName string) (string, error) {
	binPath, err := utilityBinPath()
	if err != nil {
		return "", err
	}
	absPath := filepath.Join(binPath, binName)
	_, err = os.Stat(absPath)
	if errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	return absPath, nil
}

func utilityDataPath() (string, error) {
	path := xdg.DataSubpath("devbox/util")
	return path, errors.WithStack(os.MkdirAll(path, 0o755))
}

func utilityNixProfilePath() (string, error) {
	path, err := utilityDataPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(path, "profile"), nil
}

func utilityBinPath() (string, error) {
	nixProfilePath, err := utilityNixProfilePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(nixProfilePath, "bin"), nil
}
