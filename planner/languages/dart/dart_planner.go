// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package dart

import (
	"fmt"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/planner/plansdk"
)

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "dart.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return false
	}
	return a.HasAnyFile("pubspec.yaml")
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	plan, err := p.getPlan(srcDir)
	if err != nil {
		// Lets log this
		return nil
	}
	return plan
}

func (p *Planner) getPlan(srcDir string) (*plansdk.Plan, error) {
	pubspec, err := pubspec(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return nil, err
	}

	return &plansdk.Plan{
		DevPackages: []string{"dart"},
		InstallStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    "dart pub get",
		},
		BuildStage: &plansdk.Stage{
			Command: fmt.Sprintf("dart compile exe bin/%s", pubspec.Name),
		},
		StartStage: &plansdk.Stage{
			InputFiles: []string{fmt.Sprintf("./bin/%s.exe", pubspec.Name)},
			Command:    fmt.Sprintf("./%s.exe", pubspec.Name),
		},
	}, nil
}

type Pubspec struct {
	Name string `yaml:"name,omitempty"`
}

func pubspec(srcDir string) (*Pubspec, error) {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return nil, err
	}
	paths := a.GlobFiles("pubspec.yaml")
	if len(paths) < 1 {
		return nil, errors.Errorf("expected to find a pubspec.yaml file in directory %s", srcDir)
	}
	projectFilePath := paths[0]

	pubspec := &Pubspec{}
	err = cuecfg.ParseFile(projectFilePath, pubspec)
	return pubspec, err
}
