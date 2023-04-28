// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/cuecfg"
)

const lockFileVersion = "1"

// Lightly inspired by package-lock.json
type File struct {
	devboxProject
	resolver

	LockFileVersion string             `json:"lockfile_version"`
	Packages        map[string]Package `json:"packages"`
}

type Package struct {
	LastModified string `json:"last_modified"`
	Resolved     string `json:"resolved"`
	Version      string `json:"version"`
}

func GetFile(project devboxProject, resolver resolver) (*File, error) {
	lockFile := &File{
		devboxProject: project,
		resolver:      resolver,

		LockFileVersion: lockFileVersion,
		Packages:        map[string]Package{},
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
		if l.IsVersionedPackage(p) {
			if _, err := l.Resolve(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *File) Remove(pkgs ...string) error {
	for _, p := range pkgs {
		delete(l.Packages, p)
	}
	return l.Update()
}

func (l *File) Resolve(pkg string) (string, error) {
	if _, ok := l.Packages[pkg]; !ok {
		var locked *Package
		var err error
		if l.IsVersionedPackage(pkg) {
			name, version, _ := strings.Cut(pkg, "@")
			locked, err = l.resolver.Resolve(name, version)
			if err != nil {
				return "", err
			}
		} else {
			// These are legacy packages without a version. Resolve to nixpkgs with
			// whatever hash is in the devbox.json
			locked = &Package{
				Resolved: fmt.Sprintf(
					"github:NixOS/nixpkgs/%s#%s",
					l.NixPkgsCommitHash(),
					pkg,
				),
			}
		}
		l.Packages[pkg] = *locked
		if err := l.Update(); err != nil {
			return "", err
		}
	}

	return l.Packages[pkg].Resolved, nil
}

func (l *File) Update() error {
	// Never write lockfile if versioned packages is not enabled
	if !featureflag.LockFile.Enabled() {
		return nil
	}

	return cuecfg.WriteFile(lockFilePath(l), l)
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
