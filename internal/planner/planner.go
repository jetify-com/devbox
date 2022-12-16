// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/planner/languages/c"
	"go.jetpack.io/devbox/internal/planner/languages/clojure"
	"go.jetpack.io/devbox/internal/planner/languages/cplusplus"
	"go.jetpack.io/devbox/internal/planner/languages/crystal"
	"go.jetpack.io/devbox/internal/planner/languages/dart"
	"go.jetpack.io/devbox/internal/planner/languages/deno"
	"go.jetpack.io/devbox/internal/planner/languages/dotnet"
	"go.jetpack.io/devbox/internal/planner/languages/elixir"
	"go.jetpack.io/devbox/internal/planner/languages/erlang"
	"go.jetpack.io/devbox/internal/planner/languages/fsharp"
	"go.jetpack.io/devbox/internal/planner/languages/golang"
	"go.jetpack.io/devbox/internal/planner/languages/haskell"
	"go.jetpack.io/devbox/internal/planner/languages/java"
	"go.jetpack.io/devbox/internal/planner/languages/javascript"
	"go.jetpack.io/devbox/internal/planner/languages/kotlin"
	"go.jetpack.io/devbox/internal/planner/languages/lua"
	"go.jetpack.io/devbox/internal/planner/languages/nginx"
	"go.jetpack.io/devbox/internal/planner/languages/nim"
	"go.jetpack.io/devbox/internal/planner/languages/objectivec"
	"go.jetpack.io/devbox/internal/planner/languages/ocaml"
	"go.jetpack.io/devbox/internal/planner/languages/perl"
	"go.jetpack.io/devbox/internal/planner/languages/php"
	"go.jetpack.io/devbox/internal/planner/languages/python"
	"go.jetpack.io/devbox/internal/planner/languages/ruby"
	"go.jetpack.io/devbox/internal/planner/languages/rust"
	"go.jetpack.io/devbox/internal/planner/languages/scala"
	"go.jetpack.io/devbox/internal/planner/languages/swift"
	"go.jetpack.io/devbox/internal/planner/languages/zig"
	"go.jetpack.io/devbox/internal/planner/plansdk"
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
