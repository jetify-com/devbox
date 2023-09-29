// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package python

import (
	"path/filepath"

	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/initrec/recommenders"
)

type RecommenderPip struct {
	SrcDir string
}

// implements interface recommenders.Recommender (compile-time check)
var _ recommenders.Recommender = (*RecommenderPip)(nil)

func (r *RecommenderPip) IsRelevant() bool {
	return fileutil.Exists(filepath.Join(r.SrcDir, "requirements.txt"))
}

func (r *RecommenderPip) Packages() []string {
	return []string{
		"python3",
	}
}
