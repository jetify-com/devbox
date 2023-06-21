// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/trace"

	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/auth"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
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
	overwrite bool
	url       string
}

func New(devbox devboxProject, url string, overwrite bool) *pullbox {
	return &pullbox{devbox, overwrite, url}
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
	} else if notEmpty && !p.overwrite {
		return fs.ErrExist
	}

	if p.url != "" {
		ux.Finfo(os.Stderr, "Pulling global config from %s\n", p.url)
	} else {
		ux.Finfo(os.Stderr, "Pulling global config\n")
	}

	var tmpDir string

	if p.url == "" {
		user, err := auth.GetUser()
		if err != nil {
			return err
		}
		profile := "default" // TODO: make this editable
		if tmpDir, err = s3.PullToTmp(ctx, user, profile); err != nil {
			return err
		}
		return p.copyToProfile(tmpDir)
	}

	if git.IsRepoURL(p.url) {
		if tmpDir, err = git.CloneToTmp(p.url); err != nil {
			return err
		}
		// Remove the .git directory, we don't want to keep state
		if err := os.RemoveAll(filepath.Join(tmpDir, ".git")); err != nil {
			return errors.WithStack(err)
		}
		return p.copyToProfile(tmpDir)
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

		if tmpDir, err = tar.Extract(data); err != nil {
			return err
		}

		return p.copyToProfile(tmpDir)
	}

	return usererr.New("Could not determine how to pull %s", p.url)
}

func (p *pullbox) Push(ctx context.Context) error {
	if p.url != "" {
		ux.Finfo(os.Stderr, "Pushing global config to %s\n", p.url)
	} else {
		ux.Finfo(os.Stderr, "Pushing global config\n")
	}

	if p.url == "" {
		profile := "default" // TODO: make this editable
		user, err := auth.GetUser()
		if err != nil {
			return err
		}
		ux.Finfo(
			os.Stderr,
			"Logged in as %s, pushing to to devbox cloud (profile: %s)\n",
			user.Email(),
			profile,
		)
		return s3.Push(ctx, user, p.ProjectDir(), profile)
	}
	return git.Push(ctx, p.ProjectDir(), p.url)
}
