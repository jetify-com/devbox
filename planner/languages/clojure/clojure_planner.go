// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package clojure

import "go.jetpack.io/devbox/planner/plansdk"

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "clojure.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	return false
}

func (p *Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	return &plansdk.ShellPlan{}
}

func (p *Planner) GetBuildPlan(srcDir string) *plansdk.BuildPlan {
	return &plansdk.BuildPlan{}
}
