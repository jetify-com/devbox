package haskell

import (
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

// This Project struct corresponds to the package.yaml generated during `stack new <project-name>`.
// The generated code will have stack.yaml, package.yaml and <project-name>.cabal files. This can be
// rather confusing. In short:
// - stack.yaml: has project config
// - package.yaml: has a description of the package
// - <project-name>.cabal: also has a description of the package but in "cabal file format".
const (
	packageYaml = "package.yaml"
	stackYaml   = "stack.yaml"
)

type Suggestor struct{}

// implements interface Suggestor (compile-time check)
var _ suggestors.Suggestor = (*Suggestor)(nil)

func (p *Suggestor) IsRelevant(srcDir string) bool {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return false
	}
	isRelevant := a.HasAnyFile(stackYaml)

	return isRelevant
}

func (p *Suggestor) Packages(_ string) []string {
	return []string{"stack", "libiconv", "libffi", "binutils", "ghc"}
}
