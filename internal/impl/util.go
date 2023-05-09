// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/nix"
)

// addDevboxUtilityPackage adds a package to the devbox utility profile.
// It's used to install applications devbox might need, like process-compose
// This is an alternative to a global install which would modify a user's
// environment.
func (d *Devbox) addDevboxUtilityPackage(pkg string) error {
	profilePath := fileutil.UtilityNixProfileDir
	// ensure the dir exists
	if err := os.MkdirAll(profilePath, 0755); err != nil {
		return errors.WithStack(err)
	}

	return nix.ProfileInstall(&nix.ProfileInstallArgs{
		Lockfile:    d.lockfile,
		Package:     pkg,
		ProfilePath: profilePath,
		Writer:      d.writer,
	})
}

func utilityLookPath(binName string) (string, error) {
	binPath := fileutil.UtilityBinaryDir
	// ensure the dir exists
	if err := os.MkdirAll(binPath, 0755); err != nil {
		return "", errors.WithStack(err)
	}

	absPath := filepath.Join(binPath, binName)
	_, err := os.Stat(absPath)
	if errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	return absPath, nil
}
