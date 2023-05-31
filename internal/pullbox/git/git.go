// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func CloneToTmp(repo string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "devbox")
	if err != nil {
		return "", errors.WithStack(err)
	}
	if err := clone(repo, tmpDir); err != nil {
		return "", errors.WithStack(err)
	}
	if err := os.RemoveAll(filepath.Join(tmpDir, ".git")); err != nil {
		return "", errors.WithStack(err)
	}
	return tmpDir, nil
}

func IsRepoURL(url string) bool {
	// For now only support ssh
	return strings.HasPrefix(url, "git@")
}

func clone(repo, dir string) error {
	cmd := exec.Command("git", "clone", repo, dir)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = dir
	err := cmd.Run()
	return errors.WithStack(err)
}
