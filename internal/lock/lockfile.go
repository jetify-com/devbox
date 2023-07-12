// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/searcher"

	"go.jetpack.io/devbox/internal/cuecfg"
)

const lockFileVersion = "1"

// Lightly inspired by package-lock.json
type File struct {
	devboxProject

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

func (l *File) Add(pkgs ...string) error {
	for _, p := range pkgs {
		if _, err := l.Resolve(p); err != nil {
			return err
		}
	}
	return l.Save()
}

func (l *File) Remove(pkgs ...string) error {
	for _, p := range pkgs {
		delete(l.Packages, p)
	}
	return l.Save()
}

// Resolve updates the in memory copy for performance but does not write to disk
// This avoids writing values that may need to be removed in case of error.
func (l *File) Resolve(pkg string) (*Package, error) {
	entry, hasEntry := l.Packages[pkg]

	if !hasEntry || entry.Resolved == "" {
		locked := &Package{}
		var err error
		if _, _, versioned := searcher.ParseVersionedPackage(pkg); versioned {
			locked, err = l.FetchResolvedPackage(pkg)
			if err != nil {
				return nil, err
			}
		} else if IsLegacyPackage(pkg) {
			// These are legacy packages without a version. Resolve to nixpkgs with
			// whatever hash is in the devbox.json
			locked = &Package{
				Resolved: l.LegacyNixpkgsPath(pkg),
				Source:   nixpkgSource,
			}
		}
		l.Packages[pkg] = locked
	}

	return l.Packages[pkg], nil
}

func (l *File) ForceResolve(pkg string) (*Package, error) {
	delete(l.Packages, pkg)
	return l.Resolve(pkg)
}

func (l *File) Save() error {
	return cuecfg.WriteFile(lockFilePath(l.devboxProject), l)
}

func (l *File) LegacyNixpkgsPath(pkg string) string {
	return fmt.Sprintf(
		"github:NixOS/nixpkgs/%s#%s",
		l.NixPkgsCommitHash(),
		pkg,
	)
}

func (l *File) Get(pkg string) *Package {
	entry, hasEntry := l.Packages[pkg]
	if !hasEntry || entry.Resolved == "" {
		return nil
	}
	return entry
}

func (l *File) HasAllowInsecurePackages() bool {
	for _, pkg := range l.Packages {
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
func (l *File) Tidy() {
	l.Packages = lo.PickByKeys(l.Packages, l.devboxProject.Packages())
}

func lockFilePath(project devboxProject) string {
	return filepath.Join(project.ProjectDir(), "devbox.lock")
}

func getLockfileHash(project devboxProject) (string, error) {
	return cuecfg.FileHash(lockFilePath(project))
}
