// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package zig

import (
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
