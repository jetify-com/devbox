// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package python

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/initrec/recommenders"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type RecommenderPoetry struct {
	SrcDir string
}

// implements interface recommenders.Recommender (compile-time check)
var _ recommenders.Recommender = (*RecommenderPoetry)(nil)

func (r *RecommenderPoetry) IsRelevant() bool {
	return fileutil.Exists(filepath.Join(r.SrcDir, "poetry.lock")) ||
		fileutil.Exists(filepath.Join(r.SrcDir, "pyproject.toml"))
}
func (r *RecommenderPoetry) Packages() []string {
	version := r.PythonVersion()
	pythonPkg := fmt.Sprintf("python%s", version.MajorMinorConcatenated())

	return []string{
		pythonPkg,
		"poetry",
	}
}

// TODO: This can be generalized to all python planners
func (r *RecommenderPoetry) PythonVersion() *plansdk.Version {
	defaultVersion, _ := plansdk.NewVersion("3.10.6")
	project := r.PyProject()

	if project == nil {
		return defaultVersion
	}

	if v, err := plansdk.NewVersion(project.Tool.Poetry.Dependencies.Python); err == nil {
		return v
	}
	return defaultVersion
}

type pyProject struct {
	Tool struct {
		Poetry struct {
			Name         string `toml:"name"`
			Dependencies struct {
				Python string `toml:"python"`
			} `toml:"dependencies"`
			Packages []struct {
				Include string `toml:"include"`
				From    string `toml:"from"`
			} `toml:"packages"`
			Scripts map[string]string `toml:"scripts"`
		} `toml:"poetry"`
	} `toml:"tool"`
}

func (r *RecommenderPoetry) PyProject() *pyProject {
	pyProjectPath := filepath.Join(r.SrcDir, "pyproject.toml")
	content, err := os.ReadFile(pyProjectPath)
	if err != nil {
		return nil
	}
	proj := pyProject{}
	_ = toml.Unmarshal(content, &proj)
	return &proj
}
