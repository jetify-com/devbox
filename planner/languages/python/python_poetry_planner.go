// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"go.jetpack.io/devbox/boxcli/usererr"
	"go.jetpack.io/devbox/planner/plansdk"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type PoetryPlanner struct{}

// PythonPoetryPlanner implements interface Planner (compile-time check)
var _ plansdk.Planner = (*PoetryPlanner)(nil)

func (p *PoetryPlanner) Name() string {
	return "python.Planner"
}

func (p *PoetryPlanner) IsRelevant(srcDir string) bool {
	return plansdk.FileExists(filepath.Join(srcDir, "poetry.lock")) ||
		plansdk.FileExists(filepath.Join(srcDir, "pyproject.toml"))
}

func (p *PoetryPlanner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	version := p.PythonVersion(srcDir)
	pythonPkg := fmt.Sprintf("python%s", version.MajorMinorConcatenated())

	return &plansdk.ShellPlan{
		DevPackages: []string{
			pythonPkg,
			"poetry",
		},
	}
}

func (p *PoetryPlanner) GetBuildPlan(srcDir string) *plansdk.BuildPlan {
	version := p.PythonVersion(srcDir)
	pythonPkg := fmt.Sprintf("python%s", version.MajorMinorConcatenated())
	plan := &plansdk.BuildPlan{
		DevPackages: []string{
			pythonPkg,
			"poetry",
		},
		RuntimePackages: []string{pythonPkg},
	}
	if buildable, err := p.isBuildable(srcDir); !buildable {
		return plan.WithError(err)
	}

	plan.InstallStage = &plansdk.Stage{
		InputFiles: []string{"."},
		// pex is is incompatible with certain less common python versions,
		// but because versions are sometimes expressed open-ended (e.g. ^3.10)
		// It will cause `poetry add pex` to fail. One solution is to use: --version
		// flag but when using that flag, the nix container can no longer find pex.
		Command: "poetry add pex -n --no-ansi && " +
			"poetry install --no-dev -n --no-ansi",
	}
	plan.BuildStage = &plansdk.Stage{Command: p.buildCommand(srcDir)}
	plan.StartStage = &plansdk.Stage{
		InputFiles: []string{"."},
		Command:    "python ./app.pex",
	}
	return plan
}

// TODO: This can be generalized to all python planners
func (p *PoetryPlanner) PythonVersion(srcDir string) *plansdk.Version {
	defaultVersion, _ := plansdk.NewVersion("3.10.6")
	project := p.PyProject(srcDir)

	if project == nil {
		return defaultVersion
	}

	if v, err := plansdk.NewVersion(project.Tool.Poetry.Dependencies.Python); err == nil {
		return v
	}
	return defaultVersion
}

func (p *PoetryPlanner) buildCommand(srcDir string) string {
	project := p.PyProject(srcDir)
	// Assume name follows https://peps.python.org/pep-0508/#names
	// Do simple replacement "-" -> "_" and check if any script matches name.
	// This could be improved.
	moduleName := strings.ReplaceAll(project.Tool.Poetry.Name, "-", "_")
	if _, ok := project.Tool.Poetry.Scripts[moduleName]; ok {
		// return moduleName, nil
		return p.formatBuildCommand(moduleName, moduleName)
	}
	// otherwise use the first script alphabetically
	// (go-toml doesn't preserve order, we could parse ourselves)
	scripts := maps.Keys(project.Tool.Poetry.Scripts)
	slices.Sort(scripts)
	script := ""
	if len(scripts) > 0 {
		script = scripts[0]
	}
	return p.formatBuildCommand(moduleName, script)
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

func (p *PoetryPlanner) PyProject(srcDir string) *pyProject {
	pyProjectPath := filepath.Join(srcDir, "pyproject.toml")
	content, err := os.ReadFile(pyProjectPath)
	if err != nil {
		return nil
	}
	proj := pyProject{}
	_ = toml.Unmarshal(content, &proj)
	return &proj
}

func (p *PoetryPlanner) isBuildable(srcDir string) (bool, error) {
	project := p.PyProject(srcDir)
	if project == nil {
		return false, usererr.New("Could not build container for python " +
			"application. pyproject.toml is missing and needed to install python " +
			"dependencies.")
	}

	// is this the right way to determine package name?
	packageName := strings.ReplaceAll(project.Tool.Poetry.Name, "-", "_")

	// First try to find a __main__ module as entry point
	if len(project.Tool.Poetry.Packages) > 0 {
		// If package has custom directory, check that.
		// Using packages disables auto-detection of __main__ module.
		for _, pkg := range project.Tool.Poetry.Packages {
			if pkg.Include == packageName &&
				plansdk.FileExists(filepath.Join(srcDir, pkg.From, pkg.Include, "__main__.py")) {
				return true, nil
			}
		}

		// Use setup tools auto-detect directory structure
	} else if plansdk.FileExists(filepath.Join(srcDir, packageName, "__main__.py")) ||
		plansdk.FileExists(filepath.Join(srcDir, "src", packageName, "__main__.py")) {

		return true, nil
	}

	// Fallback to using poetry scripts
	if len(project.Tool.Poetry.Scripts) == 0 {
		return false,
			usererr.New(
				"Project is not buildable: no __main__.py file found and " +
					"no scripts defined in pyproject.toml",
			)
	}
	return true, nil
}

func (p *PoetryPlanner) formatBuildCommand(module, script string) string {

	// If no scripts, just run the module directly always.
	if script == "" {
		return fmt.Sprintf(
			"poetry run pex . -o app.pex -m %s --validate-entry-point",
			module,
		)
	}

	entrypointScript := fmt.Sprintf(
		`$(poetry run python -c "import pkgutil;
import %[1]s;
modules = [name for _, name, _ in pkgutil.iter_modules(%[1]s.__path__)];
print('-m %[1]s' if '__main__' in modules else '--script %[2]s');")
`,
		module,
		script,
	)

	return fmt.Sprintf(
		"poetry run pex . -o app.pex %s --validate-entry-point &>/dev/null || "+
			"(echo 'Build failed. Could not find entrypoint' && exit 1)",
		strings.TrimSpace(strings.ReplaceAll(entrypointScript, "\n", "")),
	)
}
