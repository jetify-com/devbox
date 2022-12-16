// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package javascript

import (
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

func (p *Planner) nodeProject(srcDir string) *nodeProject {
	packageJSONPath := filepath.Join(srcDir, "package.json")
	project := &nodeProject{}
	_ = cuecfg.ParseFile(packageJSONPath, project)

	return project
}
