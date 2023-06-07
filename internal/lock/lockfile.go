// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/cuecfg"
)

const lockFileVersion = "1"

// Lightly inspired by package-lock.json
type File struct {
	project devboxProject
	resolver

	LockFileVersion string              `json:"lockfile_version"`
	Packages        map[string]*Package `json:"packages"`
}

type Package struct {
	LastModified  string `json:"last_modified,omitempty"`
	PluginVersion string `json:"plugin_version,omitempty"`
	Resolved      string `json:"resolved,omitempty"`
	Version       string `json:"version,omitempty"`
}

func GetFile(project devboxProject, resolver resolver) (*File, error) {
	lockFile := &File{
		project:  project,
		resolver: resolver,

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
	if entry, ok := l.Packages[pkg]; !ok || entry.Resolved == "" {
		locked := &Package{}
		var err error
		if IsVersionedPackage(pkg) {
			locked, err = l.resolver.Resolve(pkg)
			if err != nil {
				return nil, err
			}
		} else if IsLegacyPackage(pkg) {
			// These are legacy packages without a version. Resolve to nixpkgs with
			// whatever hash is in the devbox.json
			locked = &Package{Resolved: l.LegacyNixpkgsPath(pkg)}
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
	// Never write lockfile if versioned packages is not enabled
	if !featureflag.LockFile.Enabled() {
		return nil
	}

	return cuecfg.WriteFile(lockFilePath(l.project), l)
}

func (l *File) LegacyNixpkgsPath(pkg string) string {
	return fmt.Sprintf(
		"github:NixOS/nixpkgs/%s#%s",
		l.project.NixPkgsCommitHash(),
		pkg,
	)
}

func (l *File) Tidy(project devboxProject) {
	l.Packages = lo.PickByKeys(l.Packages, project.Packages())
}

func IsVersionedPackage(pkg string) bool {
	name, version, found := strings.Cut(pkg, "@")
	return found && name != "" && version != ""
}

// This probably belongs in input.go but can't add it there because it will
// create a circular dependency. We could move Input into own package.
func IsLegacyPackage(pkg string) bool {
	return !IsVersionedPackage(pkg) &&
		!strings.Contains(pkg, ":") &&
		// We don't support absolute paths without "path:" prefix, but adding here
		// just inc ase we ever do.
		// Landau note: I don't think we should support it, it's hard to read and a
		// bit ambiguous.
		!strings.HasPrefix(pkg, "/")
}

func lockFilePath(project devboxProject) string {
	return filepath.Join(project.ProjectDir(), "devbox.lock")
}

func getLockfileHash(project devboxProject) (string, error) {
	if !featureflag.LockFile.Enabled() {
		return "", nil
	}
	return cuecfg.FileHash(lockFilePath(project))
}

func (l *File) ConfigHash() (string, error) {
	return l.ConfigHash()
}

func (l *File) NixPkgsCommitHash() string {
	return l.NixPkgsCommitHash()
}
