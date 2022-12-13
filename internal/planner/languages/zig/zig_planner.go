// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package zig

import (
	"fmt"
	"os"
	"regexp"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "zig.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return false
	}
	return a.HasAnyFile("build.zig")
}

func (p *Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	return &plansdk.ShellPlan{
		DevPackages: []string{"zig"},
	}
}

func (p *Planner) GetBuildPlan(srcDir string) *plansdk.BuildPlan {

	var runtimePkgs []string
	var startStage *plansdk.Stage
	exeName, err := getZigExecutableName(srcDir)
	if err != nil {
		runtimePkgs = []string{"zig"}
		startStage = &plansdk.Stage{
			InputFiles: plansdk.AllFiles(),
			Command:    "zig build run",
		}
	} else {
		runtimePkgs = []string{}
		startStage = &plansdk.Stage{
			InputFiles: []string{"./zig-out/bin/"},
			Command:    fmt.Sprintf("./%s", exeName),
		}
	}

	return &plansdk.BuildPlan{
		DevPackages:     []string{"zig"},
		RuntimePackages: runtimePkgs,
		BuildStage: &plansdk.Stage{
			InputFiles: plansdk.AllFiles(),
			Command:    "zig build install",
		},
		StartStage: startStage,
	}
}

func getZigExecutableName(srcDir string) (string, error) {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return "", err
	}
	contents, err := os.ReadFile(a.AbsPath("build.zig"))
	if err != nil {
		return "", errors.WithStack(err)
	}

	r := regexp.MustCompile("addExecutable\\(\"(.*)\",.+\\)")
	matches := r.FindStringSubmatch(string(contents))
	if len(matches) != 2 {
		errorPrefix := "Unable to resolve executable name"
		if len(matches) < 2 {
			return "", errors.Errorf("%s: did not find a matching addExecutable statement", errorPrefix)
		} else {
			return "", errors.Errorf("%s: found more than one addExecutable statement", errorPrefix)
		}
	}
	return matches[1], nil

}
