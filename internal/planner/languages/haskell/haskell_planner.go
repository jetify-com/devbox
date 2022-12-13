// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package haskell

import (
	"fmt"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cuecfg"
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

func (p *Planner) GetBuildPlan(srcDir string) *plansdk.BuildPlan {
	plan, err := p.getPlan(srcDir)
	if err != nil {
		return nil
	}
	return plan
}

func (p *Planner) getPlan(srcDir string) (*plansdk.BuildPlan, error) {

	project, err := getProject(srcDir)
	if err != nil {
		return nil, err
	}

	exeName := fmt.Sprintf("%s-exe", project.Name)
	packages := []string{"stack", "libiconv", "libffi", "binutils", "ghc"}

	return &plansdk.BuildPlan{
		DevPackages:     packages,
		RuntimePackages: packages,
		InstallStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    "stack build --system-ghc --dependencies-only",
		},
		BuildStage: &plansdk.Stage{
			Command: "stack build --system-ghc",
		},
		StartStage: &plansdk.Stage{
			// The image size can be very large (> 2GB). Consider copying the binary
			// from `$(stack path --local-install-root --system-ghc)/bin`. Not doing
			// it because I haven't investigated if this would work in all scenarios.
			// Idea from: https://gist.github.com/TimWSpence/9b89b0915bf5224128e4b96abfd4ce02
			// https://medium.com/permutive/optimized-docker-builds-for-haskell-76a9808eb10b
			InputFiles: []string{"."},
			Command:    fmt.Sprintf("stack exec --system-ghc %s", exeName),
		},
	}, nil
}

func getProject(srcDir string) (*Project, error) {

	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return nil, err
	}
	paths := a.GlobFiles(packageYaml)
	if len(paths) < 1 {
		return nil, errors.Errorf(
			"expected to find a %s file in directory %s",
			packageYaml,
			srcDir,
		)
	}
	projectFilePath := paths[0]

	project := &Project{}
	err = cuecfg.ParseFile(projectFilePath, &project)
	return project, err
}
