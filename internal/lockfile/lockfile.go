package lockfile

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/cuecfg"
)

const lockFileVersion = "1"

// Lightly inspired by package-lock.json
type Lockfile struct {
	devboxProject
	resolver resolver

	LockFileVersion string                 `json:"lockfile_version"`
	Packages        map[string]PackageLock `json:"packages"`
}

type PackageLock struct {
	LastModified string `json:"last_modified"`
	Resolved     string `json:"resolved"`
	Version      string `json:"version"`
}

func Get(project devboxProject, resolver resolver) (*Lockfile, error) {
	lockFile := &Lockfile{
		devboxProject: project,
		resolver:      resolver,

		LockFileVersion: lockFileVersion,
		Packages:        map[string]PackageLock{},
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

func (l *Lockfile) Add(pkgs ...string) error {
	for _, p := range pkgs {
		if l.resolver.IsVersionedPackage(p) {
			if _, err := l.Resolve(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *Lockfile) Remove(pkgs ...string) error {
	for _, p := range pkgs {
		delete(l.Packages, p)
	}
	return l.Update()
}

func (l *Lockfile) Resolve(pkg string) (string, error) {
	if _, ok := l.Packages[pkg]; !ok {
		name, version, _ := strings.Cut(pkg, "@")
		locked, err := l.resolver.Resolve(name, version)
		if err != nil {
			return "", err
		}
		l.Packages[pkg] = *locked
		if err := l.Update(); err != nil {
			return "", err
		}
	}

	return l.Packages[pkg].Resolved, nil
}

func (l *Lockfile) Update() error {
	// Never write lockfile if versioned packages is not enabled
	if !featureflag.VersionedPackages.Enabled() {
		return nil
	}

	return cuecfg.WriteFile(lockFilePath(l), l)
}

func lockFilePath(project devboxProject) string {
	return filepath.Join(project.ProjectDir(), "devbox.lock")
}
