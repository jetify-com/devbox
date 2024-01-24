// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/internal/devpkg/pkgtype"
	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/pkg/runx/impl/types"

	"go.jetpack.io/devbox/internal/cuecfg"
)

const lockFileVersion = "1"

// Lightly inspired by package-lock.json
type File struct {
	devboxProject `json:"-"`

	LockFileVersion string `json:"lockfile_version"`

	// Packages is keyed by "canonicalName@version"
	Packages map[string]*Package `json:"packages"`
}

func GetFile(project devboxProject) (*File, error) {
	lockFile := &File{
		devboxProject: project,

		LockFileVersion: lockFileVersion,
		Packages:        map[string]*Package{},
	}
	err := cuecfg.ParseFile(lockFilePath(project), lockFile)
	if errors.Is(err, fs.ErrNotExist) {
		return lockFile, nil
	}
	if err != nil {
		return nil, err
	}
	return lockFile, nil
}

func (f *File) Add(pkgs ...string) error {
	for _, p := range pkgs {
		if _, err := f.Resolve(p); err != nil {
			return err
		}
	}
	return f.Save()
}

func (f *File) Remove(pkgs ...string) error {
	for _, p := range pkgs {
		delete(f.Packages, p)
	}
	return f.Save()
}

// Resolve updates the in memory copy for performance but does not write to disk
// This avoids writing values that may need to be removed in case of error.
func (f *File) Resolve(pkg string) (*Package, error) {
	entry, hasEntry := f.Packages[pkg]

	if !hasEntry || entry.Resolved == "" {
		locked := &Package{}
		var err error
		if _, _, versioned := searcher.ParseVersionedPackage(pkg); pkgtype.IsRunX(pkg) || versioned {
			locked, err = f.FetchResolvedPackage(pkg)
			if err != nil {
				return nil, err
			}
		} else if IsLegacyPackage(pkg) {
			// These are legacy packages without a version. Resolve to nixpkgs with
			// whatever hash is in the devbox.json
			locked = &Package{
				Resolved: f.LegacyNixpkgsPath(pkg),
				Source:   nixpkgSource,
			}
		}
		f.Packages[pkg] = locked
	}

	return f.Packages[pkg], nil
}

func (f *File) ForceResolve(pkg string) (*Package, error) {
	delete(f.Packages, pkg)
	return f.Resolve(pkg)
}

func (f *File) Save() error {
	return cuecfg.WriteFile(lockFilePath(f.devboxProject), f)
}

func (f *File) UpdateAndSaveLocalLock() error {
	return updateLocal(f.devboxProject)
}

func (f *File) LegacyNixpkgsPath(pkg string) string {
	return fmt.Sprintf(
		"github:NixOS/nixpkgs/%s#%s",
		f.NixPkgsCommitHash(),
		pkg,
	)
}

func (f *File) Get(pkg string) *Package {
	entry, hasEntry := f.Packages[pkg]
	if !hasEntry || entry.Resolved == "" {
		return nil
	}
	return entry
}

func (f *File) HasAllowInsecurePackages() bool {
	for _, pkg := range f.Packages {
		if pkg.AllowInsecure {
			return true
		}
	}
	return false
}

// This probably belongs in input.go but can't add it there because it will
// create a circular dependency. We could move Input into own package.
func IsLegacyPackage(pkg string) bool {
	_, _, versioned := searcher.ParseVersionedPackage(pkg)
	return !versioned &&
		!strings.Contains(pkg, ":") &&
		// We don't support absolute paths without "path:" prefix, but adding here
		// just in case we ever do.
		// Landau note: I don't think we should support it, it's hard to read and a
		// bit ambiguous.
		!strings.HasPrefix(pkg, "/")
}

// Tidy ensures that the lockfile has the set of packages corresponding to the devbox.json config.
// It gets rid of older packages that are no longer needed.
func (f *File) Tidy() {
	f.Packages = lo.PickByKeys(f.Packages, f.devboxProject.PackageNames())
}

// IsUpToDateAndInstalled returns true if the lockfile is up to date and the
// local hashes match, which generally indicates all packages are correctly
// installed and print-dev-env has been computed and cached.
func (f *File) IsUpToDateAndInstalled() (bool, error) {
	if dirty, err := f.isDirty(); err != nil {
		return false, err
	} else if dirty {
		return false, nil
	}
	return isLocalUpToDate(f.devboxProject)
}

func (f *File) isDirty() (bool, error) {
	currentHash, err := cachehash.JSON(f)
	if err != nil {
		return false, err
	}
	fileSystemLockFile, err := GetFile(f.devboxProject)
	if err != nil {
		return false, err
	}
	filesystemHash, err := cachehash.JSON(fileSystemLockFile)
	if err != nil {
		return false, err
	}
	return currentHash != filesystemHash, nil
}

func lockFilePath(project devboxProject) string {
	return filepath.Join(project.ProjectDir(), "devbox.lock")
}

func getLockfileHash(project devboxProject) (string, error) {
	return cachehash.JSONFile(lockFilePath(project))
}

func ResolveRunXPackage(ctx context.Context, pkg string) (types.PkgRef, error) {
	ref, err := types.NewPkgRef(strings.TrimPrefix(pkg, pkgtype.RunXPrefix))
	if err != nil {
		return types.PkgRef{}, err
	}

	registry, err := pkgtype.RunXRegistry(ctx)
	if err != nil {
		return types.PkgRef{}, err
	}
	return registry.ResolveVersion(ref)
}
