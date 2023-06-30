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
	"go.jetpack.io/devbox/internal/boxcli/featureflag"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/devpkg"
)

const lockFileVersion = "1"

// Lightly inspired by package-lock.json
type File struct {
	devboxProject
	resolver

	LockFileVersion string `json:"lockfile_version"`

	// Packages is keyed by "canonicalName@version"
	Packages map[string]*Package `json:"packages"`

}

type Package struct {
	LastModified  string `json:"last_modified,omitempty"`
	PluginVersion string `json:"plugin_version,omitempty"`
	Resolved      string `json:"resolved,omitempty"`
	Version       string `json:"version,omitempty"`
	// Systems is keyed by the system name
	Systems map[string]*SystemInfo `json:"systems,omitempty"`
}

type SystemInfo struct {
	System   string // stored elsewhere in json: it's the key for the Package.Systems
	FromHash string `json:"from_hash,omitempty"`
	// StoreName may be different from the canonicalName so we store it separately
	StoreName    string `json:"store_name,omitempty"`
	StoreVersion string `json:"store_version,omitempty"`
	ToHash       string `json:"to_hash,omitempty"`
}

func GetFile(project devboxProject, resolver resolver, system string) (*File, error) {
	lockFile := &File{
		devboxProject: project,
		resolver:      resolver,

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

	userSys, err := l.devboxProject.System()
	if err != nil {
		return nil, err
	}

	// If the package's system info is missing, we need to resolve it again.
	needsSysInfo := false
	if hasEntry && featureflag.RemoveNixpkgs.Enabled() {
		needsSysInfo = entry.Systems[userSys] == nil
	}

	if !hasEntry || entry.Resolved == "" || needsSysInfo {
		locked := &Package{}
		var err error
		if _, _, versioned := devpkg.ParseVersionedPackage(pkg); versioned {
			locked, err = l.resolver.Resolve(pkg)
			if err != nil {
				return nil, err
			}
		} else if IsLegacyPackage(pkg) {
			// These are legacy packages without a version. Resolve to nixpkgs with
			// whatever hash is in the devbox.json
			locked = &Package{Resolved: l.LegacyNixpkgsPath(pkg)}
		}

		// Merge the system info from the lockfile with the queried system info.
		// This is necessary because a different system's info may previously have
		// been added. For example: `aarch64-darwin` was already added, but
		// current user is on `x86_64-linux`.
		if hasEntry && featureflag.RemoveNixpkgs.Enabled() {
			for _, sysInfo := range entry.Systems {
				if _, ok := locked.Systems[sysInfo.System]; !ok {
					locked.Systems[sysInfo.System] = sysInfo
				}
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

func (l *File) ResolveToCurrentNixpkgCommitHash(pkg string) error {
	name, version, found := strings.Cut(pkg, "@")
	if found && version != "latest" {
		return errors.New(
			"only allowed version is @latest. Otherwise we can't guarantee the " +
				"version will resolve")
	}
	l.Packages[pkg] = &Package{Resolved: l.LegacyNixpkgsPath(name)}
	return nil
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

// This probably belongs in input.go but can't add it there because it will
// create a circular dependency. We could move Input into own package.
func IsLegacyPackage(pkg string) bool {
	_, _, versioned := devpkg.ParseVersionedPackage(pkg)
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
