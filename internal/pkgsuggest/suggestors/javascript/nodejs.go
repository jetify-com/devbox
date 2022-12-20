package javascript

import (
	"path/filepath"

	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type Suggestor struct{}

// implements interface Suggestor (compile-time check)
var _ suggestors.Suggestor = (*Suggestor)(nil)

func (s *Suggestor) IsRelevant(srcDir string) bool {
	packageJSONPath := filepath.Join(srcDir, "package.json")
	return plansdk.FileExists(packageJSONPath)
}

func (s *Suggestor) Packages(_ string) []string {

	return []string{
		"nodejs-18_x",
		"yarn",
	}
}
