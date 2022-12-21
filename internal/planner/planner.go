// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/planner/languages/php"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

var PLANNERS = []plansdk.Planner{
	&php.V2Planner{},
}

// Return a merged shell plan from shell planners if user defined packages
// contain one or more dev packages from a shell planner.
func GetShellPlan(srcDir string, userPkgs []string) *plansdk.ShellPlan {
	result := &plansdk.ShellPlan{}
	planners := getRelevantPlanners(srcDir, userPkgs)
	for _, p := range planners {
		pkgs := p.GetShellPlan(srcDir).DevPackages
		mutualPkgs := lo.Intersect(userPkgs, pkgs)
		// Only apply shell plan if user packages list all the packages from shell plan.
		if len(mutualPkgs) == len(pkgs) {
			// if merge fails, we return no errors for now.
			result, _ = plansdk.MergeShellPlans(result, p.GetShellPlan(srcDir))
		}
	}
	return result
}

// Return a merged shell plan from all planners.
func GetShellPackageSuggestion(srcDir string, userPkgs []string) []string {
	result := &plansdk.ShellPlan{}
	planners := getRelevantPlanners(srcDir, userPkgs)
	for _, p := range planners {
		result, _ = plansdk.MergeShellPlans(result, p.GetShellPlan(srcDir))
	}

	return result.DevPackages
}

func getRelevantPlanners(srcDir string, userPkgs []string) []plansdk.Planner {
	result := []plansdk.Planner{}
	for _, planner := range PLANNERS {
		if p, ok := planner.(plansdk.PlannerForPackages); ok &&
			p.IsRelevantForPackages(userPkgs) {
			result = append(result, planner)
		} else if planner.IsRelevant(srcDir) {
			result = append(result, planner)
		}
	}
	return result
}
