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

func (n *Planner) Name() string {
	return "javascript.Planner"
}

func (n *Planner) IsRelevant(srcDir string) bool {
	packageJSONPath := filepath.Join(srcDir, "package.json")
	return plansdk.FileExists(packageJSONPath)
}

func (n *Planner) GetPlan(srcDir string) *plansdk.Plan {
	packages := []string{n.nodePackage(srcDir)}
	pkgManager := "npm"
	inputFiles := []string{
		filepath.Join(srcDir, "package.json"),
	}

	npmPkgLockPath := filepath.Join(srcDir, "package-lock.json")
	if plansdk.FileExists(npmPkgLockPath) {
		inputFiles = append(inputFiles, npmPkgLockPath)
	}

	yarnPkgLockPath := filepath.Join(srcDir, "yarn.lock")
	if plansdk.FileExists(yarnPkgLockPath) {
		pkgManager = "yarn"
		packages = append(packages, "yarn")
		inputFiles = append(inputFiles, yarnPkgLockPath)
	}

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
				// Command: "" (command should be set by users. Some apps don't require a build command.)
			},

			StartStage: &plansdk.Stage{
				// Start command could be `Node server.js`, `npm serve`, `yarn start`, or anything really.
				// For now we use `node index.js` as the default.
				Command: "node index.js",
			},
		},
	}
}

type nodeProject struct {
	Engines struct {
		Node string `json:"node,omitempty"`
	} `json:"engines,omitempty"`
}

func (n *Planner) nodePackage(srcDir string) string {
	v := n.nodeVersion(srcDir)
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

func (n *Planner) nodeVersion(srcDir string) *plansdk.Version {
	p := n.nodeProject(srcDir)
	if p != nil {
		if v, err := plansdk.NewVersion(p.Engines.Node); err == nil {
			return v
		}
	}

	return nil
}

func (n *Planner) nodeProject(srcDir string) *nodeProject {
	packageJSONPath := filepath.Join(srcDir, "package.json")
	p := &nodeProject{}
	_ = cuecfg.ReadFile(packageJSONPath, p)

	return p
}
