// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nginx

import (
	"path/filepath"

	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/initrec/recommenders"
)

type Recommender struct {
	SrcDir string
}

// implements interface recommenders.Recommender (compile-time check)
var _ recommenders.Recommender = (*Recommender)(nil)

func (r *Recommender) IsRelevant() bool {
	return fileutil.Exists(filepath.Join(r.SrcDir, "nginx.conf")) ||
		fileutil.Exists(filepath.Join(r.SrcDir, "shell-nginx.conf"))
}

func (r *Recommender) Packages() []string {
	return []string{
		"nginx",
		"shell-nginx",
	}
}
