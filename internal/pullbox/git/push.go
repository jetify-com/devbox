package git

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/pullbox/ioutil"
)

const nothingToCommitErrorText = "nothing to commit"

func Push(dir, url string) error {
	tmpDir, err := CloneToTmp(url)
	if err != nil {
		return err
	}
	if err := removeNonGitFiles(tmpDir); err != nil {
		return err
	}

	if err := ioutil.CopyAll(dir, tmpDir); err != nil {
		return err
	}

	if err := createCommit(tmpDir); err != nil {
		return err
	}

	return push(tmpDir)
}

func createCommit(dir string) error {
	cmd := ioutil.CommandTTY("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return errors.WithStack(err)
	}
	cmd, buf := ioutil.CommandTTYWithBuffer(
		"git", "commit", "-m", "devbox commit")
	cmd.Dir = dir
	err := cmd.Run()
	if strings.Contains(buf.String(), nothingToCommitErrorText) {
		return nil
	}
	return errors.WithStack(err)
}

func push(dir string) error {
	cmd := ioutil.CommandTTY("git", "push")
	cmd.Dir = dir
	err := cmd.Run()
	return errors.WithStack(err)
}

func removeNonGitFiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
