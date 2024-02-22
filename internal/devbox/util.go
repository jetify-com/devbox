// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/nix/nixprofile"

	"go.jetpack.io/devbox/internal/xdg"
)

// addDevboxUtilityPackage adds a package to the devbox utility profile.
// It's used to install applications devbox might need, like process-compose
// This is an alternative to a global install which would modify a user's
// environment.
func (d *Devbox) addDevboxUtilityPackage(ctx context.Context, pkgName string) error {
	pkg := devpkg.PackageFromStringWithDefaults(pkgName, d.lockfile)
	installable, err := pkg.Installable()
	if err != nil {
		return err
	}

	profilePath, err := utilityNixProfilePath()
	if err != nil {
		return err
	}

	return nix.ProfileInstall(ctx, &nix.ProfileInstallArgs{
		Installable: installable,
		ProfilePath: profilePath,
		Writer:      d.stderr,
	})
}

func (d *Devbox) removeDevboxUtilityPackage(pkgName string) error {
	pkg := devpkg.PackageFromStringWithDefaults(pkgName, d.lockfile)
	installable, err := pkg.Installable()
	if err != nil {
		return err
	}

	utilityProfilePath, err := utilityNixProfilePath()
	if err != nil {
		return err
	}

	profile, err := nixprofile.ProfileListItems(d.stderr, utilityProfilePath)
	if err != nil {
		return err
	}

	for i, profileItem := range profile {
		if profileItem.MatchesUnlockedReference(installable) {
			return nix.ProfileRemove(utilityProfilePath, fmt.Sprint(i))
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
