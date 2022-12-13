// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package golang

import (
	"go/build"
	"os"
	"path/filepath"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/planner/plansdk"
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

func (p *Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	goPkg := getGoPackage(srcDir)

	return &plansdk.ShellPlan{
		DevPackages: []string{goPkg},
	}
}

func (p *Planner) GetBuildPlan(srcDir string) *plansdk.BuildPlan {
	goPkg := getGoPackage(srcDir)
	buildCmd, buildErr := getGoBuildCommand(srcDir)
	return &plansdk.BuildPlan{
		DevPackages: []string{
			goPkg,
		},
		InstallStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    "go get",
		},
		BuildStage: &plansdk.Stage{
			Command: buildCmd,
			Warning: buildErr,
		},
		StartStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    "./app",
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

func getGoBuildCommand(srcDir string) (string, error) {
	p, err := build.ImportDir(srcDir, build.FindOnly)
	var userError error
	if err != nil || !p.IsCommand() {
		userError = usererr.New("Cannot find main() in the directory. If you wish to specify a different import path, add `build_stage` in your devbox.json")
	}
	return "CGO_ENABLED=0 go build -o app", userError
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
	if parsed.Go == nil {
		return ""
	}
	return parsed.Go.Version
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
