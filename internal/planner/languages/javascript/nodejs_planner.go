// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package javascript

import (
	"fmt"
	"path/filepath"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/planner/plansdk"
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

func (p *Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	pkgManager := p.packageManager(srcDir)
	project := p.nodeProject(srcDir)
	packages := p.packages(pkgManager, project)

	return &plansdk.ShellPlan{
		DevPackages: packages,
	}
}

func (p *Planner) GetBuildPlan(srcDir string) *plansdk.BuildPlan {
	pkgManager := p.packageManager(srcDir)
	project := p.nodeProject(srcDir)
	packages := p.packages(pkgManager, project)
	inputFiles := p.inputFiles(srcDir)

	return &plansdk.BuildPlan{
		DevPackages: packages,
		// TODO: Optimize runtime packages to remove npm or yarn if startStage command use Node directly.
		RuntimePackages: packages,

		InstallStage: &plansdk.Stage{
			InputFiles: inputFiles,
			Command:    fmt.Sprintf("%s install", pkgManager),
		},

		BuildStage: &plansdk.Stage{
			// Copy the rest of the directory over, since at install stage we only copied package.json and its lock file.
			InputFiles: []string{"."},
			Command:    p.buildCommand(srcDir, pkgManager, project),
		},

		StartStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    p.startCommand(pkgManager, project),
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

var versionMap = map[string]string{
	// Map node versions to the corresponding nixpkgs:
	"10": "nodejs-10_x",
	"12": "nodejs-12_x",
	"16": "nodejs-16_x",
	"18": "nodejs-18_x",
}
var defaultNodeJSPkg = "nodejs"

func (p *Planner) nodePackage(project *nodeProject) string {
	v := p.nodeVersion(project)
	if v != nil {
		pkg, ok := versionMap[v.Major()]
		if ok {
			return pkg
		}
	}

	return defaultNodeJSPkg
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
		"package.json",
	}

	npmPkgLockFile := "package-lock.json"
	if plansdk.FileExists(filepath.Join(srcDir, npmPkgLockFile)) {
		inputFiles = append(inputFiles, npmPkgLockFile)
	}

	yarnPkgLockFile := "yarn.lock"
	if plansdk.FileExists(filepath.Join(srcDir, yarnPkgLockFile)) {
		inputFiles = append(inputFiles, yarnPkgLockFile)
	}

	return inputFiles
}

var buildCmdMap = map[string]string{
	// Map package manager to build command:
	"npm":  "npm run build",
	"yarn": "yarn build",
}
var postBuildCmdHookMap = map[string]string{
	// Map package manager to post build hook command:
	"npm":  "npm prune --production",
	"yarn": "yarn install --production --ignore-scripts --prefer-offline",
}

func (p *Planner) buildCommand(srcDir string, pkgManager string, project *nodeProject) string {
	buildScript := project.Scripts.Build
	if buildScript != "" {
		return fmt.Sprintf("%s && %s", buildCmdMap[pkgManager], postBuildCmdHookMap[pkgManager])
	} else {
		if p.hasTypescriptConfig(srcDir) {
			return fmt.Sprintf("%s && %s", "npx tsc", postBuildCmdHookMap[pkgManager])
		} else {
			// Still runs the post build command hook to clean up dev packages.
			return postBuildCmdHookMap[pkgManager]
		}
	}
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

func (p *Planner) hasTypescriptConfig(srcDir string) bool {
	tsPath := filepath.Join(srcDir, "tsconfig.json")
	return plansdk.FileExists(tsPath)
}
