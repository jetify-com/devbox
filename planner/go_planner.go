// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"os"
	"path/filepath"
)

type GoPlanner struct{}

// GoPlanner implements interface Planner (compile-time check)
var _ Planner = (*GoPlanner)(nil)

func (g *GoPlanner) Name() string {
	return "GoPlanner"
}

func (g *GoPlanner) IsRelevant(srcDir string) bool {
	goModPath := filepath.Join(srcDir, "go.mod")
	return fileExists(goModPath)
}

func (g *GoPlanner) GetPlan(srcDir string) *Plan {
	return &Plan{
		Packages: []string{
			"go",
		},
		InstallStage: &Stage{
			Command: "go get",
		},
		BuildStage: &Stage{
			Command: "CGO_ENABLED=0 go build -o app",
		},
		StartStage: &Stage{
			Command: "./app",
		},
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
