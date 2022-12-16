// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package python

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/internal/planner/plansdk"
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

func (p *PIPPlanner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	return &plansdk.ShellPlan{
		DevPackages: []string{
			"python3",
		},
		ShellInitHook: []string{p.shellInitHook(srcDir)},
	}
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
