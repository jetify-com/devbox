// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

type PythonPlanner struct{}

// PythonPlanner implements interface Planner (compile-time check)
var _ Planner = (*PythonPlanner)(nil)

func (g *PythonPlanner) Name() string {
	return "PythonPlanner"
}

func (g *PythonPlanner) IsRelevant(srcDir string) bool {
	return false
}

func (g *PythonPlanner) GetPlan(srcDir string) *Plan {
	return &Plan{}
}
