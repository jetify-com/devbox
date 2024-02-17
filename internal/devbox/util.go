// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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

func (d *Devbox) removeDevboxUtilityPackage(ctx context.Context, pkgName string) error {
	pkg := devpkg.PackageFromStringWithDefaults(pkgName, d.lockfile)
	installable, err := pkg.Installable()
	if err != nil {
		return err
	}

	utilProfile := nix.NixProfile{}
	utilityProfilePath, err := utilityNixProfilePath()
	if err != nil {
		return err
	}
	profileString, err := nix.ProfileList(d.stderr, utilityProfilePath, true)
	if err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(profileString), &utilProfile); err != nil {
		return err
	}

	index := -1
	// Handle utils from Nixpkgs (e.g. flake:nixpkgs#hello)
	if installable[:13] == "flake:nixpkgs" {
		installable = installable[14:]
		for i := range utilProfile.Elements {
			// check that the end of the attribute path is the same as the package name
			// These have the format "legacyPackages.<platform>.<package>", so split into 3 substrings and check the last one
			// TODO: This is hacky, find a better way.
			attrPath := strings.SplitAfterN(utilProfile.Elements[i].AttrPath, ".", 3)
			originalURL := utilProfile.Elements[i].OriginalUrl
			if attrPath[len(attrPath)-1] == installable && originalURL == "flake:nixpkgs" {
				index = i
				break
			}
		}
	} else {
		// Handle utils from other Flakes. Here we just remove the entry whose originalUrl matches the installable.
		for i := range utilProfile.Elements {
			if utilProfile.Elements[i].OriginalUrl == installable {
				index = i
				break
			}
		}

		if index >= 0 {
			if err = nix.ProfileRemove(utilityProfilePath, fmt.Sprint(index)); err != nil {
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
