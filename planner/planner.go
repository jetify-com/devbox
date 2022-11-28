// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"github.com/samber/lo"
	"go.jetpack.io/devbox/boxcli/usererr"
	"go.jetpack.io/devbox/planner/languages/c"
	"go.jetpack.io/devbox/planner/languages/clojure"
	"go.jetpack.io/devbox/planner/languages/cplusplus"
	"go.jetpack.io/devbox/planner/languages/crystal"
	"go.jetpack.io/devbox/planner/languages/dart"
	"go.jetpack.io/devbox/planner/languages/deno"
	"go.jetpack.io/devbox/planner/languages/dotnet"
	"go.jetpack.io/devbox/planner/languages/elixir"
	"go.jetpack.io/devbox/planner/languages/erlang"
	"go.jetpack.io/devbox/planner/languages/fsharp"
	"go.jetpack.io/devbox/planner/languages/golang"
	"go.jetpack.io/devbox/planner/languages/haskell"
	"go.jetpack.io/devbox/planner/languages/java"
	"go.jetpack.io/devbox/planner/languages/javascript"
	"go.jetpack.io/devbox/planner/languages/kotlin"
	"go.jetpack.io/devbox/planner/languages/lua"
	"go.jetpack.io/devbox/planner/languages/nginx"
	"go.jetpack.io/devbox/planner/languages/nim"
	"go.jetpack.io/devbox/planner/languages/objectivec"
	"go.jetpack.io/devbox/planner/languages/ocaml"
	"go.jetpack.io/devbox/planner/languages/perl"
	"go.jetpack.io/devbox/planner/languages/php"
	"go.jetpack.io/devbox/planner/languages/python"
	"go.jetpack.io/devbox/planner/languages/ruby"
	"go.jetpack.io/devbox/planner/languages/rust"
	"go.jetpack.io/devbox/planner/languages/scala"
	"go.jetpack.io/devbox/planner/languages/swift"
	"go.jetpack.io/devbox/planner/languages/zig"
	"go.jetpack.io/devbox/planner/plansdk"
)

var PLANNERS = []plansdk.Planner{
	&c.Planner{},
	&clojure.Planner{},
	&cplusplus.Planner{},
	&crystal.Planner{},
	&dotnet.Planner{},
	&dart.Planner{},
	&deno.Planner{},
	&elixir.Planner{},
	&erlang.Planner{},
	&fsharp.Planner{},
	&golang.Planner{},
	&haskell.Planner{},
	&java.Planner{},
	&javascript.Planner{},
	&kotlin.Planner{},
	&lua.Planner{},
	&nginx.Planner{},
	&nim.Planner{},
	&objectivec.Planner{},
	&ocaml.Planner{},
	&perl.Planner{},
	&php.Planner{},
	&php.V2Planner{},
	&python.PoetryPlanner{},
	&python.PIPPlanner{},
	&ruby.Planner{},
	&rust.Planner{},
	&scala.Planner{},
	&swift.Planner{},
	&zig.Planner{},
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

// Return one buildable plan from all planners.
// If no buildable plan is found, return errors from the first unbuildable plan.
func GetBuildPlan(srcDir string, userPkgs []string) (*plansdk.BuildPlan, error) {
	buildables := []*plansdk.BuildPlan{}
	unbuildables := []*plansdk.BuildPlan{}
	for _, p := range getRelevantPlanners(srcDir, userPkgs) {
		plan := p.GetBuildPlan(srcDir)
		mutualPkgs := lo.Intersect(userPkgs, plan.DevPackages)
		if len(mutualPkgs) > 0 {
			if !plan.Invalid() {
				buildables = append(buildables, plan)
			} else {
				unbuildables = append(unbuildables, plan)
			}
		}
	}
	// If we could not find any buildable plans, and at least one unbuildable plan,
	// Let's let the user know. Question: How should we handle multiple
	// unbuildable plans?
	if len(buildables) == 0 && len(unbuildables) > 0 {
		if err := unbuildables[0].Error(); err != nil {
			return nil, err
		}
		return nil, usererr.New("Unable to build project")
	}
	if len(buildables) > 1 {
		// TODO(Landau) Ideally we give the user a way to resolve this
		return nil, usererr.New("Multiple buildable plans found: %v", buildables)
	}
	if len(buildables) == 0 {
		return nil, usererr.New(
			"Devbox could not find a buildable plan for this project. If your " +
				"project/language is currently supported, please create an issue at " +
				"https://github.com/jetpack-io/devbox/issues - if it's not supported " +
				"you can request it!",
		)
	}
	return buildables[0], nil
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
