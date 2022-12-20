package zig

import (
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type Suggestor struct{}

// implements interface Suggestor (compile-time check)
var _ suggestors.Suggestor = (*Suggestor)(nil)

func (s *Suggestor) IsRelevant(srcDir string) bool {

	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return false
	}
	return a.HasAnyFile("build.zig")
}

func (s *Suggestor) Packages(_ string) []string {
	return []string{
		"zig",
	}
}
