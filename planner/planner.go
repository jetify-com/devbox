// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import "go.jetpack.io/devbox/boxcli/usererr"

type Planner interface {
	Name() string
	IsRelevant(srcDir string) bool
	GetPlan(srcDir string) *Plan
}

var PLANNERS = []Planner{
	&GoPlanner{},
	&PythonPoetryPlanner{},
}

func GetPlan(srcDir string) *Plan {
	result := &Plan{
		DevPackages:     []string{},
		RuntimePackages: []string{},
	}
	for _, planner := range getRelevantPlans(srcDir) {
		result = MergePlans(result, planner.GetPlan(srcDir))
	}
	return result
}

func HasPlan(srcDir string) bool {
	return len(getRelevantPlans(srcDir)) > 0
}

func IsBuildable(srcDir string) (bool, error) {
	buildables := []Planner{}
	for _, planner := range getRelevantPlans(srcDir) {
		if plan := planner.GetPlan(srcDir); !plan.Buildable() {
			if err := plan.Error(); err != nil {
				return false, err
			}
			return false, usererr.New("Unable to build project")
		}
		buildables = append(buildables, planner)
	}
	if len(buildables) > 1 {
		// TODO(Landau) Ideally we give the user a way to resolve this
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
