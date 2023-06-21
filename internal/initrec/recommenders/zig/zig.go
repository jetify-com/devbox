// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package zig

import (
	"go.jetpack.io/devbox/internal/initrec/analyzer"
	"go.jetpack.io/devbox/internal/initrec/recommenders"
)

type Recommender struct {
	SrcDir string
}

// implements interface recommenders.Recommender (compile-time check)
var _ recommenders.Recommender = (*Recommender)(nil)

func (r *Recommender) IsRelevant() bool {
	a, err := analyzer.NewAnalyzer(r.SrcDir)
	if err != nil {
		// We should log that an error has occurred.
		return false
	}
	return a.HasAnyFile("build.zig")
}

func (r *Recommender) Packages() []string {
	return []string{
		"zig",
	}
}
