// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

type Planner interface {
	Name() string
	IsRelevant(srcDir string) bool
	GetPlan(srcDir string) *Plan // TODO: this should probably return (*Plan, error)
}

var PLANNERS = []Planner{
	&GoPlanner{},
	&PythonPoetryPlanner{},
}

func GetPlan(srcDir string) *Plan {
	result := &Plan{
		Packages: []string{},
	}
	for _, planner := range PLANNERS {
		if planner.IsRelevant(srcDir) {
			plan := planner.GetPlan(srcDir)
			result = MergePlans(result, plan)
		}
	}
	return result
}
