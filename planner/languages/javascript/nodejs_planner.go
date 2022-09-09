// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package javascript

import (
	"fmt"
	"path/filepath"

	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/planner/plansdk"
)

type Planner struct{}

// NodeJsPlanner implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "javascript.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	packageJSONPath := filepath.Join(srcDir, "package.json")
	return plansdk.FileExists(packageJSONPath)
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	pkgManager := p.packageManager(srcDir)
	project := p.nodeProject(srcDir)
	packages := p.packages(pkgManager, project)
	inputFiles := p.inputFiles(srcDir)

	return &plansdk.Plan{
		DevPackages: packages,
		// TODO: Optimize runtime packages to remove npm or yarn if startStage command use Node directly.
		RuntimePackages: packages,

		SharedPlan: plansdk.SharedPlan{
			InstallStage: &plansdk.Stage{
				InputFiles: inputFiles,
				Command:    fmt.Sprintf("%s install", pkgManager),
			},

			BuildStage: &plansdk.Stage{
				// Copy the rest of the directory over, since at install stage we only copied package.json and its lock file.
				InputFiles: []string{"."},
				Command:    p.buildCommand(pkgManager, project),
			},

			StartStage: &plansdk.Stage{
				Command: p.startCommand(pkgManager, project),
			},
		},
	}
}

type nodeProject struct {
	Scripts struct {
		Build string `json:"build,omitempty"`
		Start string `json:"start,omitempty"`
	}
	Engines struct {
		Node string `json:"node,omitempty"`
	} `json:"engines,omitempty"`
}

func (p *Planner) nodePackage(project *nodeProject) string {
	v := p.nodeVersion(project)
	if v != nil {
		switch v.Major() {
		case "10":
			return "nodejs-10_x"
		case "12":
			return "nodejs-12_x"
		case "16":
			return "nodejs-16_x"
		case "18":
			return "nodejs-18_x"
		}
	}

	return "nodejs"
}

func (p *Planner) nodeVersion(project *nodeProject) *plansdk.Version {
	if p != nil {
		if v, err := plansdk.NewVersion(project.Engines.Node); err == nil {
			return v
		}
	}

	return nil
}

func (p *Planner) packageManager(srcDir string) string {
	yarnPkgLockPath := filepath.Join(srcDir, "yarn.lock")
	if plansdk.FileExists(yarnPkgLockPath) {
		return "yarn"
	}
	return "npm"
}

func (p *Planner) packages(pkgManager string, project *nodeProject) []string {
	nodeJSPkg := p.nodePackage(project)
	pkgs := []string{nodeJSPkg}

	if pkgManager == "yarn" {
		return append(pkgs, "yarn")
	}
	return pkgs
}

func (p *Planner) inputFiles(srcDir string) []string {
	inputFiles := []string{
		filepath.Join(srcDir, "package.json"),
	}

	npmPkgLockPath := filepath.Join(srcDir, "package-lock.json")
	if plansdk.FileExists(npmPkgLockPath) {
		inputFiles = append(inputFiles, npmPkgLockPath)
	}

	yarnPkgLockPath := filepath.Join(srcDir, "yarn.lock")
	if plansdk.FileExists(yarnPkgLockPath) {
		inputFiles = append(inputFiles, yarnPkgLockPath)
	}

	return inputFiles
}

func (p *Planner) buildCommand(pkgManager string, project *nodeProject) string {
	buildScript := project.Scripts.Build
	postBuildCmdHook := "npm prune --production"

	if pkgManager == "yarn" {
		postBuildCmdHook = "yarn install --production --ignore-scripts --prefer-offline"
	}
	if buildScript == "" {
		return postBuildCmdHook
	}

	return fmt.Sprintf("%s build && %s", pkgManager, postBuildCmdHook)
}

func (p *Planner) startCommand(pkgManager string, project *nodeProject) string {
	startScript := project.Scripts.Start
	if startScript == "" {
		// Start command could be `Node server.js`, `npm serve`, or anything really.
		// For now we use `node index.js` as the default.
		return "node index.js"
	}

	return fmt.Sprintf("%s start", pkgManager)
}

func (p *Planner) nodeProject(srcDir string) *nodeProject {
	packageJSONPath := filepath.Join(srcDir, "package.json")
	project := &nodeProject{}
	_ = cuecfg.ParseFile(packageJSONPath, project)

	return project
}
