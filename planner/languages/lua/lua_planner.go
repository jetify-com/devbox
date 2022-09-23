// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lua

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/planner/plansdk"
)

const rockspecExtension = "*.rockspec"

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "lua.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return false
	}
	isRelevant := a.HasAnyFile(rockspecExtension)
	return isRelevant
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	plan, err := p.getPlan(srcDir)
	if err != nil {
		return nil
	}
	return plan
}

func (p *Planner) getPlan(srcDir string) (*plansdk.Plan, error) {
	absRockspecFile, err := getRockspecFile(srcDir)
	if err != nil {
		return nil, err
	}

	absSrcDir, err := filepath.Abs(srcDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get abs from srcDir: %s", srcDir)
	}
	// rockspecFile := absRockspecFile
	rockspecFile, err := filepath.Rel(absSrcDir, absRockspecFile)
	if err != nil {
		fmt.Printf("err is %s\n", err)
		return nil, errors.Wrapf(err, "failed to get rel path from %s", srcDir)
	}
	fmt.Sprintf("rockspec file %s\n", rockspecFile)

	packages := []string{"lua", "luarocks"}
	return &plansdk.Plan{
		DevPackages:     packages,
		RuntimePackages: packages,
		InstallStage: &plansdk.Stage{
			InputFiles: plansdk.AllFiles(),
			Command: fmt.Sprintf(
				"luarocks --tree devbox-luarocks install --only"+
					"-deps %s", rockspecFile,
			),
		},
		BuildStage: &plansdk.Stage{
			Command: "luarocks --tree devbox-luarocks build",
		},
		StartStage: &plansdk.Stage{
			InputFiles: plansdk.AllFiles(),

			// TODO savil. What is this command?
			// Command: "luarocks --tree devbox-luarocks <something here>",
			Command: strings.Join(
				[]string{
					"eval $(luarocks --tree devbox-luarocks path)",
					"lua main.lua",
				},
				" && ",
			),
		},
	}, nil
}

func getRockspecFile(srcDir string) (string, error) {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		return "", err
	}

	filepaths := a.GlobFiles(rockspecExtension)
	if len(filepaths) != 1 {
		return "", errors.Errorf("expected exactly one .rockspec file but received: %s", strings.Join(filepaths, ","))
	}
	return filepaths[0], nil
}
