// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
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
	poetryLockPath := filepath.Join(srcDir, "poetry.lock")
	mainPYPath := filepath.Join(srcDir, "main.py")
	return fileExists(poetryLockPath) && fileExists(mainPYPath)
}

func (g *PythonPoetryPlanner) GetPlan(srcDir string) *Plan {
	version := g.PythonVersion(srcDir)
	return &Plan{
		Packages: []string{
			fmt.Sprintf("python%s", version.majorMinorConcatenated()),
			"poetry",
		},
		InstallStage: &Stage{
			Command: "poetry add pex -D -n --no-ansi && poetry install --no-dev -n --no-ansi",
		},
		BuildStage: &Stage{
			Command: "poetry run pex . -o app.pex --script " + g.GetEntrypoint(srcDir),
		},
		// TODO parse pyproject.toml to get the start command?
		StartStage: &Stage{
			Command: "PEX_ROOT=/tmp/.pex python ./app.pex",
			Image:   "al3xos/python-distroless:3.10-debian11-debug",
		},
	}
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

func (g *PythonPoetryPlanner) GetEntrypoint(srcDir string) string {
	p := g.PyProject(srcDir)
	if p == nil {
		panic("pyproject.toml not found")
	}
	if len(p.Tool.Poetry.Scripts) == 0 {
		// This error message as a panic is not ideal. We should change GetPlan
		// to return (plan, error) and print a nicer formatted error message.
		panic(
			"\n\nno scripts found in pyproject.toml. Please define a script to use as " +
				"an entrypoint for your app:\n" +
				"[tool.poetry.scripts]\nmy_app = \"my_app:my_function\"\n",
		)
	}
	// Assume name follows https://peps.python.org/pep-0508/#names
	// Do simple replacement "-" -> "_" and check if any script matches name.
	// This could be improved.
	module_name := strings.ReplaceAll(p.Tool.Poetry.Name, "-", "_")
	if _, ok := p.Tool.Poetry.Scripts[module_name]; ok {
		fmt.Println("ENTRYPOINT", module_name)
		return module_name
	}
	// otherwise use the first script alphabetically
	// (go-toml doesn't preserve order, we could parse ourselves)
	scripts := maps.Keys(p.Tool.Poetry.Scripts)
	slices.Sort(scripts)
	fmt.Println("ENTRYPOINT", scripts[0])
	return scripts[0]
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
