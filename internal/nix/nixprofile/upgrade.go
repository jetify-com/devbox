// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nixprofile

import (
	"context"
	"os"

	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
)

func ProfileUpgrade(profileDir string, pkg *devpkg.Package, lock *lock.File) error {
	idx, err := ProfileListIndex(
		context.TODO(),
		&ProfileListIndexArgs{
			Lockfile:   lock,
			Writer:     os.Stderr,
			Package:    pkg,
			ProfileDir: profileDir,
		},
	)
	if err != nil {
		return err
	}

	return nix.ProfileUpgrade(profileDir, idx)
}
