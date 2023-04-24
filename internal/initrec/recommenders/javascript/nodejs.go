// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package javascript

import (
	"path/filepath"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/initrec/recommenders"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type Recommender struct {
	SrcDir string
}

// implements interface recommenders.Recommender (compile-time check)
var _ recommenders.Recommender = (*Recommender)(nil)

func (r *Recommender) IsRelevant() bool {
	packageJSONPath := filepath.Join(r.SrcDir, "package.json")
	return fileutil.Exists(packageJSONPath)
}

func (r *Recommender) Packages() []string {
	pkgManager := r.packageManager()
	project := r.nodeProject()
	packages := r.packages(pkgManager, project)

	return packages
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

func (r *Recommender) nodePackage(project *nodeProject) string {
	v := r.nodeVersion(project)
	if v != nil {
		pkg, ok := versionMap[v.Major()]
		if ok {
			return pkg
		}
	}

	return defaultNodeJSPkg
}

func (r *Recommender) nodeVersion(project *nodeProject) *plansdk.Version {
	if r != nil {
		if v, err := plansdk.NewVersion(project.Engines.Node); err == nil {
			return v
		}
	}

	return nil
}

func (r *Recommender) packageManager() string {
	yarnPkgLockPath := filepath.Join(r.SrcDir, "yarn.lock")
	if fileutil.Exists(yarnPkgLockPath) {
		return "yarn"
	}
	return "npm"
}

func (r *Recommender) packages(pkgManager string, project *nodeProject) []string {
	nodeJSPkg := r.nodePackage(project)
	pkgs := []string{nodeJSPkg}

	if pkgManager == "yarn" {
		return append(pkgs, "yarn")
	}
	return pkgs
}

func (r *Recommender) nodeProject() *nodeProject {
	packageJSONPath := filepath.Join(r.SrcDir, "package.json")
	project := &nodeProject{}
	_ = cuecfg.ParseFile(packageJSONPath, project)

	return project
}
