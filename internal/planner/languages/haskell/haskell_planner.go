// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package haskell

import (
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

const (
	packageYaml = "package.yaml"
	stackYaml   = "stack.yaml"
)

// This Project struct corresponds to the package.yaml generated during `stack new <project-name>`.
// The generated code will have stack.yaml, package.yaml and <project-name>.cabal files. This can be
// rather confusing. In short:
// - stack.yaml: has project config
// - package.yaml: has a description of the package
// - <project-name>.cabal: also has a description of the package but in "cabal file format".
//
// Cabal is an older build system for Haskell, while Stack is more modern, so I think Stack wraps over Cabal.
type Project struct {
	Name string `yaml:"name,omitempty"`
}

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "haskell.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return false
	}
	isRelevant := a.HasAnyFile(stackYaml)

	return isRelevant
}

func (p *Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	return &plansdk.ShellPlan{
		DevPackages: []string{"stack", "libiconv", "libffi", "binutils", "ghc"},
	}
}
