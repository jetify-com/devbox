// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package mutagenbox

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cloud/mutagen"
)

const (
	// relative to user home i.e. ~
	dataDirPath = ".config/devbox/mutagen"
)

// TerminateSessionsForMachine is a devbox-specific API that calls the generic mutagen terminate API.
// It relies on the mutagen-sync-session's labels to identify which sessions to terminate for
// a particular machine (fly VM).
func TerminateSessionsForMachine(machineID string, userEnv map[string]string) error {
	labels := DefaultSyncLabels(machineID)

	env, err := DefaultEnv()
	if err != nil {
		return err
	}
	// the user-specified env-vars get precedence over the defaultEnvVars
	for k, v := range userEnv {
		env[k] = v
	}

	return mutagen.Terminate(env, labels)
}

func DefaultSyncLabels(machineID string) map[string]string {
	return map[string]string{
		"devbox-vm": machineID,
	}
}

func DefaultEnv() (map[string]string, error) {
	shimDir, err := ShimDir()
	if err != nil {
		return nil, err
	}

	mutagenDir, err := createAndGetDataDir()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"MUTAGEN_SSH_PATH":       shimDir,
		"MUTAGEN_DATA_DIRECTORY": mutagenDir,
	}, nil
}

// createAndGetDataDir prepares the data directory for devbox's mutagen instance
func createAndGetDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WithStack(err)
	}

	path := filepath.Join(home, dataDirPath)
	return path, errors.WithStack(os.MkdirAll(path, 0o700))
}
