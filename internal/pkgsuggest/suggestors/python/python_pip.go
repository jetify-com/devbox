package python

import (
	"path/filepath"

	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type SuggestorPip struct{}

// implements interface Suggestor (compile-time check)
var _ suggestors.Suggestor = (*SuggestorPip)(nil)

func (s *SuggestorPip) IsRelevant(srcDir string) bool {
	return plansdk.FileExists(filepath.Join(srcDir, "requirements.txt"))
}
func (s *SuggestorPip) Packages(srcDir string) []string {

	return []string{
		"python3",
	}
}
