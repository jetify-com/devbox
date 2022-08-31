// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import "path/filepath"

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
	return &Plan{
		Packages: []string{
			"python310",
			"poetry",
		},
		InstallStage: &Stage{
			Command: "poetry install --no-dev --no-interaction --no-ansi",
		},
		BuildStage: &Stage{
			Command: "poetry build",
		},
		// TODO parse pyproject.toml to get the start command?
		StartStage: &Stage{
			Command: "python main.py",
		},
	}
}
