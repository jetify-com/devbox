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

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/pullbox/git"
	"go.jetpack.io/devbox/internal/pullbox/s3"
	"go.jetpack.io/devbox/internal/pullbox/tar"
	"go.jetpack.io/devbox/internal/ux"
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

	notEmpty, err := profileIsNotEmpty(p.ProjectDir())
	if err != nil {
		return err
	} else if notEmpty && !p.Overwrite {
		return fs.ErrExist
	}

	if p.URL != "" {
		ux.Finfo(os.Stderr, "Pulling global config from %s\n", p.URL)
	} else {
		ux.Finfo(os.Stderr, "Pulling global config\n")
	}

	var tmpDir string

	if p.URL == "" {
		if p.Credentials.IDToken == "" {
			return usererr.New("Not logged in")
		}
		profile := "default" // TODO: make this editable
		if tmpDir, err = s3.PullToTmp(ctx, &p.Credentials, profile); err != nil {
			return err
		}
		return p.copyToProfile(tmpDir)
	}

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
	if p.URL != "" {
		ux.Finfo(os.Stderr, "Pushing global config to %s\n", p.URL)
	} else {
		ux.Finfo(os.Stderr, "Pushing global config\n")
	}

	if p.URL == "" {
		profile := "default" // TODO: make this editable
		if p.Credentials.IDToken == "" {
			return usererr.New("Not logged in")
		}
		ux.Finfo(
			os.Stderr,
			"Logged in as %s, pushing to to devbox cloud (profile: %s)\n",
			p.Credentials.Email,
			profile,
		)
		return s3.Push(ctx, &p.Credentials, p.ProjectDir(), profile)
	}
	return git.Push(ctx, p.ProjectDir(), p.URL)
}
