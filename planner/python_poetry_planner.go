// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"go.jetpack.io/devbox/boxcli/usererr"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type PythonPoetryPlanner struct{}

// PythonPoetryPlanner implements interface Planner (compile-time check)
var _ Planner = (*PythonPoetryPlanner)(nil)

func (g *PythonPoetryPlanner) Name() string {
	return "PythonPoetryPlanner"
}

func (g *PythonPoetryPlanner) IsRelevant(srcDir string) bool {
	return fileExists(filepath.Join(srcDir, "poetry.lock"))
}

func (g *PythonPoetryPlanner) GetPlan(srcDir string) *Plan {
	version := g.PythonVersion(srcDir)
	plan := &Plan{
		DevPackages: []string{
			fmt.Sprintf("python%s", version.majorMinorConcatenated()),
			"poetry",
		},
	}
	if buildable, err := g.isBuildable(srcDir); !buildable {
		return plan.WithError(err)
	}
	entrypoint, err := g.GetEntrypoint(srcDir)
	if err != nil {
		return plan.WithError(err)
	}
	plan.InstallStage = &Stage{
		// pex is is incompatible with certain less common python versions,
		// but because versions are sometimes expressed open-ended (e.g. ^3.10)
		// It will cause `poetry add pex` to fail. One solution is to use: --version
		// flag but when using that flag, the nix container can no longer find pex.
		Command: "poetry add pex -n --no-ansi && " +
			"poetry install --no-dev -n --no-ansi",
	}
	plan.BuildStage = &Stage{
		Command: "PEX_ROOT=/tmp/.pex poetry run pex . -o app.pex --script " + entrypoint,
	}
	plan.StartStage = &Stage{
		Command: "PEX_ROOT=/tmp/.pex python ./app.pex",
		Image:   getPythonImage(version),
	}
	return plan
}

// TODO: This can be generalized to all python planners
func (g *PythonPoetryPlanner) PythonVersion(srcDir string) *version {
	defaultVersion, _ := newVersion("3.10.6")
	p := g.PyProject(srcDir)

	if p == nil {
		return defaultVersion
	}

	if v, err := newVersion(p.Tool.Poetry.Dependencies.Python); err == nil {
		return v
	}
	return defaultVersion
}

func (g *PythonPoetryPlanner) GetEntrypoint(srcDir string) (string, error) {
	project := g.PyProject(srcDir)
	// Assume name follows https://peps.python.org/pep-0508/#names
	// Do simple replacement "-" -> "_" and check if any script matches name.
	// This could be improved.
	moduleName := strings.ReplaceAll(project.Tool.Poetry.Name, "-", "_")
	if _, ok := project.Tool.Poetry.Scripts[moduleName]; ok {
		return moduleName, nil
	}
	// otherwise use the first script alphabetically
	// (go-toml doesn't preserve order, we could parse ourselves)
	scripts := maps.Keys(project.Tool.Poetry.Scripts)
	slices.Sort(scripts)
	return scripts[0], nil
}

type pyProject struct {
	Tool struct {
		Poetry struct {
			Name         string `toml:"name"`
			Dependencies struct {
				Python string `toml:"python"`
			} `toml:"dependencies"`
			Scripts map[string]string `toml:"scripts"`
		} `toml:"poetry"`
	} `toml:"tool"`
}

func (g *PythonPoetryPlanner) PyProject(srcDir string) *pyProject {
	pyProjectPath := filepath.Join(srcDir, "pyproject.toml")
	content, err := os.ReadFile(pyProjectPath)
	if err != nil {
		return nil
	}
	p := pyProject{}
	_ = toml.Unmarshal(content, &p)
	return &p
}

func getPythonImage(version *version) string {
	if version.exact() == "3" {
		return "al3xos/python-distroless:3.10-debian11-debug"
	}
	if version.majorMinor() == "3.10" || version.majorMinor() == "3.9" {
		return fmt.Sprintf("al3xos/python-distroless:%s-debian11-debug", version.majorMinor())
	}
	return fmt.Sprintf("python:%s-slim", version.exact())
}

func (g *PythonPoetryPlanner) isBuildable(srcDir string) (bool, error) {
	project := g.PyProject(srcDir)
	if project == nil {
		return false, usererr.New("Could not build container for python " +
			"application. pyproject.toml is missing and needed to install python " +
			"dependencies.")
	}
	if len(project.Tool.Poetry.Scripts) == 0 {
		return false,
			usererr.New("Project is not buildable: no scripts found in " +
				"pyproject.toml. Please define a script to use as an entrypoint for " +
				"your app:\n\n[tool.poetry.scripts]\nmy_app = \"my_app:my_function\"\n")
	}
	return true, nil
}
