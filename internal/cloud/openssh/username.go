// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package openssh

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/fileutil"
)

func GithubUsernameFromLocalFile() (string, error) {
	filePath, err := usernameFilePath()
	if err != nil {
		return "", err
	}
	if !fileutil.Exists(filePath) {
		return "", nil
	}

	username, err := os.ReadFile(filePath)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(username), nil
}

func SaveGithubUsernameToLocalFile(username string) error {
	filePath, err := usernameFilePath()
	if err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(os.WriteFile(filePath, []byte(username), 0o600))
}

func usernameFilePath() (string, error) {
	sshDir, err := devboxSSHDir()
	if err != nil {
		return "", errors.WithStack(err)
	}

	return filepath.Join(sshDir, "github_username"), nil
}
