// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package nginx

import (
	"path/filepath"

	"go.jetpack.io/devbox/internal/initrec/recommenders"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type Recommender struct {
	SrcDir string
}

// implements interface Recommender (compile-time check)
var _ recommenders.Recommender = (*Recommender)(nil)

func (r *Recommender) IsRelevant() bool {
	return plansdk.FileExists(filepath.Join(r.SrcDir, "nginx.conf")) ||
		plansdk.FileExists(filepath.Join(r.SrcDir, "shell-nginx.conf"))
}

func (r *Recommender) Packages() []string {
	return []string{
		"nginx",
		"shell-nginx",
	}
}
