// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type PythonPoetryPlanner struct{}

// PythonPoetryPlanner implements interface Planner (compile-time check)
var _ Planner = (*PythonPoetryPlanner)(nil)

func (g *PythonPoetryPlanner) Name() string {
	return "PythonPoetryPlanner"
}

func (g *PythonPoetryPlanner) IsRelevant(srcDir string) bool {
	poetryLockPath := filepath.Join(srcDir, "poetry.lock")
	mainPYPath := filepath.Join(srcDir, "main.py")
	return fileExists(poetryLockPath) && fileExists(mainPYPath)
}

func (g *PythonPoetryPlanner) GetPlan(srcDir string) *Plan {
	version := g.PythonVersion(srcDir)
	return &Plan{
		Packages: []string{
			fmt.Sprintf("python%s", version.majorMinorConcatenated()),
			"poetry",
		},
		InstallStage: &Stage{
			Command: "poetry install --no-dev --no-interaction --no-ansi",
		},
		BuildStage: &Stage{
			Command: "poetry build",
		},
		// TODO parse pyproject.toml to get the start command?
		StartStage: &EntrypointStage{
			Entrypoint:     "python main.py",
			Image:          fmt.Sprintf("python:%s-alpine", version.exact()),
			PrepareCommand: "pip install dist/*.whl",
		},
	}
}

// TODO: This can be generalized to all python planners
func (g *PythonPoetryPlanner) PythonVersion(srcDir string) *version {
	defaultVersion, _ := newVersion("3.10.6")
	pyProjectPath := filepath.Join(srcDir, "pyproject.toml")
	c, err := ioutil.ReadFile(pyProjectPath)
	if err != nil {
		return defaultVersion
	}
	pyProject := struct {
		Tool struct {
			Poetry struct {
				Dependencies struct {
					Python string `toml:"python"`
				} `toml:"dependencies"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}{}
	_ = toml.Unmarshal(c, &pyProject)

	if v, err := newVersion(pyProject.Tool.Poetry.Dependencies.Python); err == nil {
		return v
	}
	return defaultVersion
}
