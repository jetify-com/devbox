// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package python

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type PoetryPlanner struct{}

// PythonPoetryPlanner implements interface Planner (compile-time check)
var _ plansdk.Planner = (*PoetryPlanner)(nil)

func (p *PoetryPlanner) Name() string {
	return "python.Planner"
}

func (p *PoetryPlanner) IsRelevant(srcDir string) bool {
	return plansdk.FileExists(filepath.Join(srcDir, "poetry.lock")) ||
		plansdk.FileExists(filepath.Join(srcDir, "pyproject.toml"))
}

func (p *PoetryPlanner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	version := p.PythonVersion(srcDir)
	pythonPkg := fmt.Sprintf("python%s", version.MajorMinorConcatenated())

	return &plansdk.ShellPlan{
		DevPackages: []string{
			pythonPkg,
			"poetry",
		},
	}
}

// TODO: This can be generalized to all python planners
func (p *PoetryPlanner) PythonVersion(srcDir string) *plansdk.Version {
	defaultVersion, _ := plansdk.NewVersion("3.10.6")
	project := p.PyProject(srcDir)

	if project == nil {
		return defaultVersion
	}

	if v, err := plansdk.NewVersion(project.Tool.Poetry.Dependencies.Python); err == nil {
		return v
	}
	return defaultVersion
}

type pyProject struct {
	Tool struct {
		Poetry struct {
			Name         string `toml:"name"`
			Dependencies struct {
				Python string `toml:"python"`
			} `toml:"dependencies"`
			Packages []struct {
				Include string `toml:"include"`
				From    string `toml:"from"`
			} `toml:"packages"`
			Scripts map[string]string `toml:"scripts"`
		} `toml:"poetry"`
	} `toml:"tool"`
}

func (p *PoetryPlanner) PyProject(srcDir string) *pyProject {
	pyProjectPath := filepath.Join(srcDir, "pyproject.toml")
	content, err := os.ReadFile(pyProjectPath)
	if err != nil {
		return nil
	}
	proj := pyProject{}
	_ = toml.Unmarshal(content, &proj)
	return &proj
}
