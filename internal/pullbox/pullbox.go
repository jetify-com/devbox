// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

import (
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

func (p *pullbox) Pull() error {
	if git.IsRepoURL(p.url) {
		tmpDir, err := git.CloneToTmp(p.url)
		if err != nil {
			return err
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
