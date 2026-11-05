// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"go.jetify.com/devbox/internal/build"
	"go.jetify.com/devbox/internal/cachehash"
	"go.jetify.com/devbox/internal/cuecfg"
)

var ignoreShellMismatch = false

// stateHashFile is a non-shared lock file that helps track the state of the
// local devbox environment. It contains hashes that may not be the same across
// machines (e.g. manifest hash).
// When we do implement a shared lock file, it may contain some shared fields
// with this one but not all.
type stateHashFile struct {
	// ConfigHash covers every input of the devbox environment.
	ConfigHash string `json:"config_hash"`
	// NixConfigHash covers only the inputs that affect the Nix environment
	// (the generated flake, the print-dev-env output and the nix profile).
	// When only ConfigHash changed, the Nix work can be skipped. It's empty
	// in state files written by older versions of Devbox.
	NixConfigHash string `json:"nix_config_hash"`
	DevboxVersion string `json:"devbox_version"`
	// fish has different generated scripts so we need to recompute them if user
	// changes shell.
	IsFish                 bool   `json:"is_fish"`
	LockFileHash           string `json:"lock_file_hash"`
	NixPrintDevEnvHash     string `json:"nix_print_dev_env_hash"`
	NixProfileManifestHash string `json:"nix_profile_manifest_hash"`
}

type UpdateStateHashFileArgs struct {
	ProjectDir    string
	ConfigHash    string
	NixConfigHash string
	// IsFish is an arg because in the future we may allow the user
	// to specify shell in devbox.json which should be passed in here.
	IsFish bool
}

func UpdateAndSaveStateHashFile(args UpdateStateHashFileArgs) error {
	newLock, err := getCurrentStateHash(args)
	if err != nil {
		return err
	}

	return cuecfg.WriteFile(stateHashFilePath(args.ProjectDir), newLock)
}

// SetIgnoreShellMismatch is used to disable the shell comparison when checking
// if the state is up to date. This is useful when we don't load shellrc (e.g. running)
func SetIgnoreShellMismatch(ignore bool) {
	ignoreShellMismatch = ignore
}

// StateStatus describes how the local devbox state (.devbox/state.json)
// compares to the current project.
type StateStatus int

const (
	// StateUpToDate means nothing changed since the state was last saved.
	StateUpToDate StateStatus = iota
	// StateNonNixStale means only inputs that don't affect the Nix
	// environment changed (e.g. env, init_hook or scripts in devbox.json).
	// The generated flake, the cached print-dev-env output and the nix
	// profile are still valid, so only the non-Nix outputs need regenerating.
	StateNonNixStale
	// StateStale means the Nix environment needs to be recomputed.
	StateStale
)

func (s StateStatus) String() string {
	switch s {
	case StateUpToDate:
		return "up-to-date"
	case StateNonNixStale:
		return "non-nix-stale"
	case StateStale:
		return "stale"
	default:
		return fmt.Sprintf("StateStatus(%d)", int(s))
	}
}

func stateStatus(args UpdateStateHashFileArgs) (StateStatus, error) {
	filesystemStateHash, err := readStateHashFile(args.ProjectDir)
	if err != nil {
		return StateStale, err
	}
	newStateHash, err := getCurrentStateHash(args)
	if err != nil {
		return StateStale, err
	}

	if ignoreShellMismatch {
		filesystemStateHash.IsFish = newStateHash.IsFish
	}

	return compareStateHashes(filesystemStateHash, newStateHash), nil
}

// compareStateHashes compares the saved state with the current one.
func compareStateHashes(saved, current *stateHashFile) StateStatus {
	if *saved == *current {
		return StateUpToDate
	}
	// Everything except ConfigHash must match. A different lockfile,
	// print-dev-env cache, nix profile, Devbox version or shell all need a
	// full recompute. An empty NixConfigHash means the state was written by
	// an older Devbox that didn't record it.
	if saved.NixConfigHash == "" {
		return StateStale
	}
	savedWithCurrentConfig := *saved
	savedWithCurrentConfig.ConfigHash = current.ConfigHash
	if savedWithCurrentConfig == *current {
		return StateNonNixStale
	}
	return StateStale
}

func readStateHashFile(projectDir string) (*stateHashFile, error) {
	hashFile := &stateHashFile{}
	err := cuecfg.ParseFile(stateHashFilePath(projectDir), hashFile)
	if errors.Is(err, fs.ErrNotExist) {
		return hashFile, nil
	}
	if err != nil {
		return nil, err
	}
	return hashFile, nil
}

func getCurrentStateHash(args UpdateStateHashFileArgs) (*stateHashFile, error) {
	nixHash, err := manifestHash(args.ProjectDir)
	if err != nil {
		return nil, err
	}

	printDevEnvCacheHash, err := printDevEnvCacheHash(args.ProjectDir)
	if err != nil {
		return nil, err
	}

	lockfileHash, err := getLockfileHash(args.ProjectDir)
	if err != nil {
		return nil, err
	}

	newLock := &stateHashFile{
		ConfigHash:             args.ConfigHash,
		NixConfigHash:          args.NixConfigHash,
		DevboxVersion:          build.Version,
		IsFish:                 args.IsFish,
		LockFileHash:           lockfileHash,
		NixPrintDevEnvHash:     printDevEnvCacheHash,
		NixProfileManifestHash: nixHash,
	}

	return newLock, nil
}

func stateHashFilePath(projectDir string) string {
	return filepath.Join(projectDir, ".devbox", "state.json")
}

func manifestHash(profileDir string) (string, error) {
	return cachehash.JSONFile(filepath.Join(profileDir, ".devbox/nix/profile/default/manifest.json"))
}

func printDevEnvCacheHash(profileDir string) (string, error) {
	return cachehash.JSONFile(filepath.Join(profileDir, ".devbox/.nix-print-dev-env-cache"))
}

func getLockfileHash(projectDir string) (string, error) {
	return cachehash.JSONFile(lockFilePath(projectDir))
}
