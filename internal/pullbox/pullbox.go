// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

import (
	"context"
	"os"
	"path/filepath"
	"runtime/trace"

	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/pullbox/git"
)

type devboxProject interface {
	ProjectDir() string
}

type pullbox struct {
	devboxProject
	overwrite bool
	url       string
}

func New(devbox devboxProject, url string, overwrite bool) *pullbox {
	return &pullbox{devbox, overwrite, url}
}

func (p *pullbox) Pull(ctx context.Context) error {
	defer trace.StartRegion(ctx, "Pull").End()

	if git.IsRepoURL(p.url) {
		tmpDir, err := git.CloneToTmp(p.url)
		if err != nil {
			return err
		}
		// Remove the .git directory, we don't want to keep state
		if err := os.RemoveAll(filepath.Join(tmpDir, ".git")); err != nil {
			return errors.WithStack(err)
		}
		return p.copy(p.overwrite, tmpDir, p.ProjectDir())
	}

	if p.IsTextDevboxConfig() {
		return p.pullTextDevboxConfig()
	}

	if isArchive, err := urlIsArchive(p.url); err != nil {
		return err
	} else if isArchive {
		data, err := download(p.url)
		if err != nil {
			return err
		}
		tmpDir, err := extract(data)
		if err != nil {
			return err
		}

		return p.copy(p.overwrite, tmpDir, p.ProjectDir())
	}

	return usererr.New("Could not determine how to pull %s", p.url)
}

func (p *pullbox) Push(ctx context.Context) error {
	return git.Push(ctx, p.ProjectDir(), p.url)
}
