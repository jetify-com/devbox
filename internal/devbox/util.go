// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/nix"

	"go.jetpack.io/devbox/internal/xdg"
)

// addDevboxUtilityPackage adds a package to the devbox utility profile.
// It's used to install applications devbox might need, like process-compose
// This is an alternative to a global install which would modify a user's
// environment.
func (d *Devbox) addDevboxUtilityPackage(ctx context.Context, pkgName string) error {
	pkg := devpkg.PackageFromStringWithDefaults(pkgName, d.lockfile)
	installables, err := pkg.Installables()
	if err != nil {
		return err
	}
	profilePath, err := utilityNixProfilePath()
	if err != nil {
		return err
	}

	for _, installable := range installables {

		err = nix.ProfileInstall(ctx, &nix.ProfileInstallArgs{
			Installable: installable,
			ProfilePath: profilePath,
			Writer:      d.stderr,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Devbox) removeDevboxUtilityPackage(
	ctx context.Context, pkgName string,
) error {
	pkg := devpkg.PackageFromStringWithDefaults(pkgName, d.lockfile)
	installables, err := pkg.Installables()
	if err != nil {
		return err
	}

	utilityProfilePath, err := utilityNixProfilePath()
	if err != nil {
		return err
	}

	for _, installable := range installables {
		storePaths, err := nix.StorePathsFromInstallable(ctx, installable, false)
		if err != nil {
			return err
		}

		for _, storePath := range storePaths {
			if err = nix.ProfileRemove(utilityProfilePath, storePath); err != nil {
				return err
			}
		}
	}
	return nil
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
