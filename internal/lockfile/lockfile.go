package lockfile

import (
	"errors"
	"os"
	"path/filepath"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/nix"
)

// localLockFile is a non-shared lock file that helps track the state of the
// local devbox environment. It contains hashes that may not be the same across
// machines (e.g. manifest hash).
// When we do implement a shared lock file, it may contain some shared fields
// with this one but not all.
type localLockFile struct {
	project                devboxProject
	ConfigHash             string `json:"config_hash"`
	NixProfileManifestHash string `json:"nix_profile_manifest_hash"`
}

func (l *localLockFile) equals(other *localLockFile) bool {
	return l.ConfigHash == other.ConfigHash &&
		l.NixProfileManifestHash == other.NixProfileManifestHash
}

func (l *localLockFile) IsUpToDate() (bool, error) {
	newLock, err := forProject(l.project)
	if err != nil {
		return false, err
	}

	return l.equals(newLock), nil
}

func (l *localLockFile) Update() error {
	newLock, err := forProject(l.project)
	if err != nil {
		return err
	}
	*l = *newLock

	return cuecfg.WriteFile(localLockFilePath(l.project), l)
}

type devboxProject interface {
	ConfigHash() (string, error)
	ProjectDir() string
}

func Local(project devboxProject) (*localLockFile, error) {
	lockFile := &localLockFile{project: project}
	err := cuecfg.ParseFile(localLockFilePath(project), lockFile)
	if errors.Is(err, os.ErrNotExist) {
		return lockFile, nil
	} else if err != nil {
		return nil, err
	}
	return lockFile, nil
}

func forProject(project devboxProject) (*localLockFile, error) {
	configHash, err := project.ConfigHash()
	if err != nil {
		return nil, err
	}

	nixHash, err := nix.ManifestHash(project.ProjectDir())
	if err != nil {
		return nil, err
	}

	newLock := &localLockFile{
		project:                project,
		ConfigHash:             configHash,
		NixProfileManifestHash: nixHash,
	}

	return newLock, nil
}

func localLockFilePath(project devboxProject) string {
	return filepath.Join(project.ProjectDir(), ".devbox", "local.lock")
}
