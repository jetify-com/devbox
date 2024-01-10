// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/internal/cuecfg"
)

// localLockFile is a non-shared lock file that helps track the state of the
// local devbox environment. It contains hashes that may not be the same across
// machines (e.g. manifest hash).
// When we do implement a shared lock file, it may contain some shared fields
// with this one but not all.
type localLockFile struct {
	project                devboxProject
	ConfigHash             string `json:"config_hash"`
	DevboxVersion          string `json:"devbox_version"`
	LockFileHash           string `json:"lock_file_hash"`
	NixProfileManifestHash string `json:"nix_profile_manifest_hash"`
	NixPrintDevEnvHash     string `json:"nix_print_dev_env_hash"`
}

func (l *localLockFile) equals(other *localLockFile) bool {
	return l.ConfigHash == other.ConfigHash &&
		l.LockFileHash == other.LockFileHash &&
		l.NixProfileManifestHash == other.NixProfileManifestHash &&
		l.NixPrintDevEnvHash == other.NixPrintDevEnvHash &&
		l.DevboxVersion == other.DevboxVersion
}

func isLocalUpToDate(project devboxProject) (bool, error) {
	filesystemLock, err := readLocal(project)
	if err != nil {
		return false, err
	}
	newLock, err := forProject(project)
	if err != nil {
		return false, err
	}

	return filesystemLock.equals(newLock), nil
}

func updateLocal(project devboxProject) error {
	l, err := readLocal(project)
	if err != nil {
		return err
	}
	newLock, err := forProject(l.project)
	if err != nil {
		return err
	}
	*l = *newLock

	return cuecfg.WriteFile(localLockFilePath(l.project), l)
}

func readLocal(project devboxProject) (*localLockFile, error) {
	lockFile := &localLockFile{project: project}
	err := cuecfg.ParseFile(localLockFilePath(project), lockFile)
	if errors.Is(err, fs.ErrNotExist) {
		return lockFile, nil
	}
	if err != nil {
		return nil, err
	}
	return lockFile, nil
}

func removeLocal(project devboxProject) error {
	// RemoveAll to avoid error in case file does not exist.
	return os.RemoveAll(localLockFilePath(project))
}

func forProject(project devboxProject) (*localLockFile, error) {
	configHash, err := project.ConfigHash()
	if err != nil {
		return nil, err
	}

	nixHash, err := manifestHash(project.ProjectDir())
	if err != nil {
		return nil, err
	}

	printDevEnvCacheHash, err := printDevEnvCacheHash(project.ProjectDir())
	if err != nil {
		return nil, err
	}

	lockfileHash, err := getLockfileHash(project)
	if err != nil {
		return nil, err
	}

	newLock := &localLockFile{
		project:                project,
		ConfigHash:             configHash,
		DevboxVersion:          build.Version,
		LockFileHash:           lockfileHash,
		NixProfileManifestHash: nixHash,
		NixPrintDevEnvHash:     printDevEnvCacheHash,
	}

	return newLock, nil
}

func localLockFilePath(project devboxProject) string {
	return filepath.Join(project.ProjectDir(), ".devbox", "local.lock")
}

func manifestHash(profileDir string) (string, error) {
	return cachehash.JSONFile(filepath.Join(profileDir, ".devbox/nix/profile/default/manifest.json"))
}

func printDevEnvCacheHash(profileDir string) (string, error) {
	return cachehash.JSONFile(filepath.Join(profileDir, ".devbox/.nix-print-dev-env-cache"))
}
