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
	&PHPPlanner{},
	&PythonPoetryPlanner{},
	&NodeJSPlanner{},
}

func GetPlan(srcDir string) *Plan {
	result := &Plan{
		DevPackages:     []string{},
		RuntimePackages: []string{},
	}
	for _, p := range getRelevantPlanners(srcDir) {
		result = MergePlans(result, p.GetPlan(srcDir))
	}
	return result
}

func IsBuildable(srcDir string) (bool, error) {
	buildables := []*Plan{}
	unbuildables := []*Plan{}
	for _, p := range getRelevantPlanners(srcDir) {
		if plan := p.GetPlan(srcDir); plan.Buildable() {
			buildables = append(buildables, plan)
		} else {
			unbuildables = append(unbuildables, plan)
		}
	}
	// If we could not find any buildable plans, and at least one unbuildable plan,
	// Let's let the user know. Question: How should we handle multiple
	// unbuildable plans?
	if len(buildables) == 0 && len(unbuildables) > 0 {
		if err := unbuildables[0].Error(); err != nil {
			return false, err
		}
		return false, usererr.New("Unable to build project")
	}
	if len(buildables) > 1 {
		// TODO(Landau) Ideally we give the user a way to resolve this
		return false, usererr.New("Multiple buildable plans found: %v", buildables)
	}
	return true, nil
}

func getRelevantPlanners(srcDir string) []Planner {
	result := []Planner{}
	for _, planner := range PLANNERS {
		if planner.IsRelevant(srcDir) {
			result = append(result, planner)
		}
	}
	return result
}
