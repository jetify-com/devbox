// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package haskell

import (
	"go.jetpack.io/devbox/internal/initrec/recommenders"
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
	isRelevant := a.HasAnyFile(stackYaml)

	return isRelevant
}

func (r *Recommender) Packages() []string {
	return []string{"stack", "libiconv", "libffi", "binutils", "ghc"}
}
