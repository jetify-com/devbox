package python

import (
	"path/filepath"

	"go.jetpack.io/devbox/internal/initrec/recommenders"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type RecommenderPip struct {
	SrcDir string
}

// implements interface Recommender (compile-time check)
var _ recommenders.Recommender = (*RecommenderPip)(nil)

func (r *RecommenderPip) IsRelevant() bool {
	return plansdk.FileExists(filepath.Join(r.SrcDir, "requirements.txt"))
}
func (r *RecommenderPip) Packages() []string {

	return []string{
		"python3",
	}
}
