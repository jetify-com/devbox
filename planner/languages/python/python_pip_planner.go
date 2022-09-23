// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package python

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/boxcli/usererr"
	"go.jetpack.io/devbox/planner/plansdk"
)

// TODO: Doesn't work with libraries like Pandas that have C extensions
// We get error
// ImportError: libstdc++.so.6: cannot open shared object file: No such file or directory
// possible solution is to set $LD_LIBRARY_PATH
// https://nixos.wiki/wiki/Packaging/Quirks_and_Caveats
type PIPPlanner struct{}

// PythonPoetryPlanner implements interface Planner (compile-time check)
var _ plansdk.Planner = (*PIPPlanner)(nil)

func (p *PIPPlanner) Name() string {
	return "python.Planner"
}

func (p *PIPPlanner) IsRelevant(srcDir string) bool {
	return plansdk.FileExists(filepath.Join(srcDir, "requirements.txt"))
}
func (p *PIPPlanner) GetPlan(srcDir string) *plansdk.Plan {
	plan := &plansdk.Plan{
		DevPackages: []string{
			"python3",
		},
		RuntimePackages: []string{
			`python3`,
		},
		ShellInitHook: p.shellInitHook(srcDir),
	}
	if err := p.isBuildable(srcDir); err != nil {
		return plan.WithError(err)
	}
	plan.InstallStage = &plansdk.Stage{
		Command:    "python -m venv .venv && source .venv/bin/activate && pip install -r requirements.txt",
		InputFiles: plansdk.AllFiles(),
	}
	plan.BuildStage = &plansdk.Stage{Command: pipBuildCommand}
	plan.StartStage = &plansdk.Stage{
		Command:    "python ./app.pex",
		InputFiles: []string{"app.pex"},
	}
	return plan
}

func (p *PIPPlanner) isBuildable(srcDir string) error {
	if plansdk.FileExists(filepath.Join(srcDir, "setup.py")) {
		return nil
	}

	return usererr.New(
		"setup.py not found. Please create a setup.py file to build your project." +
			" The distribution name must be a case-insensitive match of the package" +
			" (dir) name. Dashes are converted to underscores.",
	)
}

func (p *PIPPlanner) shellInitHook(srcDir string) string {
	venvPath := filepath.Join(srcDir, ".venv")
	venvActivatePath := filepath.Join(srcDir, ".venv", "bin", "activate")
	script := strings.TrimSpace(`
echo "Creating/Using virtual environment in %[1]s";
python -m venv "%[1]s";
source "%[2]s";`)
	return fmt.Sprintf(script, venvPath, venvActivatePath)
}

var pipBuildCommand = strings.TrimSpace(`
source .venv/bin/activate && \
pip install pex && \
PACKAGE_NAME=$(python setup.py --name |  tr '[:upper:]-' '[:lower:]_') && \
pex . -o app.pex -m $PACKAGE_NAME -r requirements.txt
`,
)
