// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package git

import (
	"context"
	"runtime/trace"
	"strings"

	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/cmdutil"
	"go.jetpack.io/devbox/internal/fileutil"
)

const nothingToCommitErrorText = "nothing to commit"

func Push(ctx context.Context, dir, url string) error {
	defer trace.StartRegion(ctx, "Push").End()

	tmpDir, err := fileutil.CreateDevboxTempDir()
	if err != nil {
		return err
	}

	if err := cloneGitHistory(url, tmpDir); err != nil {
		return err
	}

	if err := fileutil.CopyAll(dir, tmpDir); err != nil {
		return err
	}

	if err := createCommit(tmpDir); err != nil {
		return err
	}

	return push(tmpDir)
}

func cloneGitHistory(url, dst string) error {
	// See https://stackoverflow.com/questions/38999901/clone-only-the-git-directory-of-a-git-repo
	cmd := cmdutil.CommandTTY("git", "clone", "--no-checkout", url, dst)
	cmd.Dir = dst
	return errors.WithStack(cmd.Run())
}

func createCommit(dir string) error {
	cmd := cmdutil.CommandTTY("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return errors.WithStack(err)
	}
	cmd, buf := cmdutil.CommandTTYWithBuffer(
		"git", "commit", "-m", "devbox commit")
	cmd.Dir = dir
	err := cmd.Run()
	if strings.Contains(buf.String(), nothingToCommitErrorText) {
		return nil
	}
	return errors.WithStack(err)
}

func push(dir string) error {
	cmd := cmdutil.CommandTTY("git", "push")
	cmd.Dir = dir
	err := cmd.Run()
	return errors.WithStack(err)
}
