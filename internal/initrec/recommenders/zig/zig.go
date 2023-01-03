package zig

import (
	"go.jetpack.io/devbox/internal/initrec/recommenders"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type Recommender struct {
	SrcDir string
}

// implements interface Recommender (compile-time check)
var _ recommenders.Recommender = (*Recommender)(nil)

func (r *Recommender) IsRelevant() bool {

	a, err := plansdk.NewAnalyzer(r.SrcDir)
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
