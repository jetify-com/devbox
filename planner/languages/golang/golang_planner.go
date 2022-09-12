// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package golang

import (
	"os"
	"path/filepath"

	"go.jetpack.io/devbox/planner/plansdk"
	"golang.org/x/mod/modfile"
)

type Planner struct{}

var versionMap = map[string]string{
	// Map go versions to the corresponding nixpkgs:
	"1.19": "go_1_19",
	"1.18": "go",
	"1.17": "go_1_17",
}

const defaultPkg = "go_1_19" // Default to "latest" for cases where we can't determine a version.

// GoPlanner implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "golang.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	goModPath := filepath.Join(srcDir, "go.mod")
	return fileExists(goModPath)
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	goPkg := getGoPackage(srcDir)
	return &plansdk.Plan{
		DevPackages: []string{
			goPkg,
		},
		SharedPlan: plansdk.SharedPlan{
			InstallStage: &plansdk.Stage{
				InputFiles: []string{"."},
				Command:    "go get",
			},
			BuildStage: &plansdk.Stage{
				Command: "CGO_ENABLED=0 go build -o app",
			},
			StartStage: &plansdk.Stage{
				InputFiles: []string{"."},
				Command:    "./app",
			},
		},
	}
}

func getGoPackage(srcDir string) string {
	goModPath := filepath.Join(srcDir, "go.mod")
	goVersion := parseGoVersion(goModPath)
	v, ok := versionMap[goVersion]
	if ok {
		return v
	} else {
		// Should we be throwing an error instead, if we don't have a nix package
		// for the specified version of go?
		return defaultPkg
	}
}

func parseGoVersion(gomodPath string) string {
	content, err := os.ReadFile(gomodPath)
	if err != nil {
		return ""
	}
	parsed, err := modfile.ParseLax(gomodPath, content, nil)
	if err != nil {
		return ""
	}
	return parsed.Go.Version
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
