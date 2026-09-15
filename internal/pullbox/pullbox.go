// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/trace"

	"github.com/pkg/errors"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/pullbox/git"
	"go.jetify.com/devbox/internal/pullbox/tar"
	"go.jetify.com/devbox/internal/ux"
)

type devboxProject interface {
	ProjectDir() string
}

type pullbox struct {
	devboxProject
	devopt.PullboxOpts
}

func New(devbox devboxProject, opts devopt.PullboxOpts) *pullbox {
	return &pullbox{devbox, opts}
}

// Pull
// This can be rewritten to be more readable and less repetitive. Possibly
// something like:
// puller := getPullerForURL(url)
// return puller.Pull()
func (p *pullbox) Pull(ctx context.Context) error {
	defer trace.StartRegion(ctx, "Pull").End()
	var err error

	if p.URL == "" {
		return usererr.New("Nothing to pull from. Pass a git repo, URL, or file path to pull from.")
	}

	notEmpty, err := profileIsNotEmpty(p.ProjectDir())
	if err != nil {
		return err
	} else if notEmpty && !p.Overwrite {
		return fs.ErrExist
	}

	ux.Finfof(os.Stderr, "Pulling global config from %s\n", p.URL)

	var tmpDir string

	if git.IsRepoURL(p.URL) {
		if tmpDir, err = git.CloneToTmp(p.URL); err != nil {
			return err
		}
		// Remove the .git directory, we don't want to keep state
		if err := os.RemoveAll(filepath.Join(tmpDir, ".git")); err != nil {
			return errors.WithStack(err)
		}
		return p.copyToProfile(tmpDir)
	}

	if p.IsTextDevboxConfig() {
		return p.pullTextDevboxConfig(ctx)
	}

	if isArchive, err := urlIsArchive(p.URL); err != nil {
		return err
	} else if isArchive {
		data, err := download(p.URL)
		if err != nil {
			return err
		}

		if tmpDir, err = tar.Extract(data); err != nil {
			return err
		}

		return p.copyToProfile(tmpDir)
	}

	return usererr.New("Could not determine how to pull %s", p.URL)
}

func (p *pullbox) Push(ctx context.Context) error {
	if p.URL == "" {
		return usererr.New("Nowhere to push to. Pass a git repo to push to.")
	}
	ux.Finfof(os.Stderr, "Pushing global config to %s\n", p.URL)
	return git.Push(ctx, p.ProjectDir(), p.URL)
}
