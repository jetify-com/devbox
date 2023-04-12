package lockfile

import (
	"errors"
	"os"
	"path/filepath"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/nix"
)

type lockFile struct {
	project                devboxProject
	ConfigHash             string `json:"config_hash"`
	NixProfileManifestHash string `json:"nix_profile_manifest_hash"`
}

func (l *lockFile) equals(other *lockFile) bool {
	return l.ConfigHash == other.ConfigHash &&
		l.NixProfileManifestHash == other.NixProfileManifestHash
}

func (l *lockFile) IsUpToDate() (bool, error) {
	newLock, err := forProject(l.project)
	if err != nil {
		return false, err
	}

	return l.equals(newLock), nil
}

type devboxProject interface {
	ConfigHash() (string, error)
	ProjectDir() string
}

func Update(proj devboxProject) error {
	newLock, err := forProject(proj)
	if err != nil {
		return err
	}

	if lock, err := Get(proj); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	} else if lock != nil && lock.equals(newLock) {
		return nil
	}

	return cuecfg.WriteFile(lockFilePath(proj), newLock)
}

func Get(project devboxProject) (*lockFile, error) {
	lockFile := &lockFile{project: project}
	err := cuecfg.ParseFile(lockFilePath(project), lockFile)
	if errors.Is(err, os.ErrNotExist) {
		return lockFile, nil
	} else if err != nil {
		return nil, err
	}
	return lockFile, nil
}

func forProject(project devboxProject) (*lockFile, error) {
	configHash, err := project.ConfigHash()
	if err != nil {
		return nil, err
	}

	nixHash, err := nix.ManifestHash(project.ProjectDir())
	if err != nil {
		return nil, err
	}

	newLock := &lockFile{
		project:                project,
		ConfigHash:             configHash,
		NixProfileManifestHash: nixHash,
	}

	return newLock, nil
}

func lockFilePath(project devboxProject) string {
	return filepath.Join(project.ProjectDir(), ".devbox", "devbox.lock")
}
