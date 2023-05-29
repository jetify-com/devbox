// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package git

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const alreadyExists = "already exists and is not an empty directory"
const uncommittedChanges = "Please commit your changes or stash them before you merge"

var ErrExist = errors.New(alreadyExists)
var ErrUncommittedChanges = errors.New("uncommitted changes")

func CloneOrPull(repo, dir string, overwrite bool) error {
	if isDirRepoRoot(dir) {
		return pull(dir, overwrite)
	}
	return clone(repo, dir, overwrite)
}

func isDirRepoRoot(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func IsRepoURL(url string) bool {
	// For now only support ssh
	return strings.HasPrefix(url, "git@")
}

func clone(repo, dir string, overwrite bool) error {
	if overwrite {
		_ = os.RemoveAll(dir)
		_ = os.MkdirAll(dir, 0755)
	}

	cmd := exec.Command("git", "clone", repo, dir)
	buf := bytes.NewBuffer(nil)
	cmd.Stderr = io.MultiWriter(os.Stderr, buf)
	cmd.Stdout = os.Stdout
	cmd.Dir = dir
	err := cmd.Run()
	if strings.Contains(buf.String(), alreadyExists) {
		return ErrExist
	}
	return errors.WithStack(err)
}

func pull(dir string, overwrite bool) error {
	if overwrite {
		if err := reset(dir); err != nil {
			return err
		}
	}
	cmd := exec.Command("git", "pull")
	buf := bytes.NewBuffer(nil)
	cmd.Stderr = io.MultiWriter(os.Stderr, buf)
	cmd.Stdout = os.Stdout
	cmd.Dir = dir
	err := cmd.Run()
	if strings.Contains(buf.String(), uncommittedChanges) {
		return ErrUncommittedChanges
	}
	return errors.WithStack(err)
}

func reset(dir string) error {
	cmd := exec.Command("git", "reset", "--hard")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = dir
	return errors.WithStack(cmd.Run())
}
