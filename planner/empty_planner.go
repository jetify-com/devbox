// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

type EmptyPlanner struct{}

// EmptyPlanner implements interface Planner (compile-time check)
var _ Planner = (*EmptyPlanner)(nil)

func (g *EmptyPlanner) Name() string {
	return "EmptyPlanner"
}

func (g *EmptyPlanner) IsRelevant(srcDir string) bool {
	return false
}

func (g *EmptyPlanner) GetPlan(srcDir string) *Plan {
	return &Plan{}
}
