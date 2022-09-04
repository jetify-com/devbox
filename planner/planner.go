// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import "go.jetpack.io/devbox/boxcli/usererr"

type Planner interface {
	Name() string
	// IsBuildable returns true if the planner can build the project.
	// It assumes that IsRelevant() has already returned true.
	IsBuildable(srcDir string) (bool, error)
	IsRelevant(srcDir string) bool
	GetPlan(srcDir string) (*Plan, error)
}

var PLANNERS = []Planner{
	&GoPlanner{},
	&PythonPoetryPlanner{},
}

func GetPlan(srcDir string) (*Plan, error) {
	result := &Plan{
		Packages: []string{},
	}
	for _, planner := range getRelevantPlans(srcDir) {
		plan, err := planner.GetPlan(srcDir)
		if err != nil {
			return nil, err
		}
		result = MergePlans(result, plan)
	}
	return result, nil
}

func HasPlan(srcDir string) bool {
	return len(getRelevantPlans(srcDir)) > 0
}

func IsBuildable(srcDir string) (bool, error) {
	buildables := []Planner{}
	for _, planner := range getRelevantPlans(srcDir) {
		if ok, err := planner.IsBuildable(srcDir); !ok {
			return false, err
		}
		buildables = append(buildables, planner)
	}
	if len(buildables) > 1 {
		return false, usererr.New("Multiple buildable plans found: %v", buildables)
	}
	return true, nil
}

func getRelevantPlans(srcDir string) []Planner {
	result := []Planner{}
	for _, planner := range PLANNERS {
		if planner.IsRelevant(srcDir) {
			result = append(result, planner)
		}
	}
	return result
}
