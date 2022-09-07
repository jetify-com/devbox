// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

type Planner interface {
	Name() string
	IsRelevant(srcDir string) bool
	GetPlan(srcDir string) (*Plan, error)
}

var PLANNERS = []Planner{
	&GoPlanner{},
	&PythonPoetryPlanner{},
}

func GetPlan(srcDir string) (*Plan, error) {
	result := &Plan{
		DevPackages:     []string{},
		RuntimePackages: []string{},
	}
	for _, planner := range PLANNERS {
		if planner.IsRelevant(srcDir) {
			plan, err := planner.GetPlan(srcDir)
			if err != nil {
				return nil, err
			}
			result = MergePlans(result, plan)
		}
	}
	return result, nil
}

func HasPlan(srcDir string) bool {
	for _, planner := range PLANNERS {
		if planner.IsRelevant(srcDir) {
			return true
		}
	}
	return false
}
