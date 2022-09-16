// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
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
	"go.jetpack.io/devbox/planner/languages/typescript"
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
	&nim.Planner{},
	&objectivec.Planner{},
	&ocaml.Planner{},
	&perl.Planner{},
	&php.Planner{},
	&python.Planner{},
	&ruby.Planner{},
	&rust.Planner{},
	&scala.Planner{},
	&swift.Planner{},
	&typescript.Planner{},
	&zig.Planner{},
}

func GetPlan(srcDir string) (*plansdk.Plan, error) {
	result := &plansdk.Plan{
		DevPackages:     []string{},
		RuntimePackages: []string{},
	}
	var err error
	for _, p := range getRelevantPlanners(srcDir) {
		result, err = plansdk.MergePlans(result, p.GetPlan(srcDir))
		if err != nil {
			return nil, err
		}

	}
	return result, nil
}

func IsBuildable(srcDir string) (bool, error) {
	buildables := []*plansdk.Plan{}
	unbuildables := []*plansdk.Plan{}
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

func getRelevantPlanners(srcDir string) []plansdk.Planner {
	result := []plansdk.Planner{}
	for _, planner := range PLANNERS {
		if planner.IsRelevant(srcDir) {
			result = append(result, planner)
		}
	}
	return result
}
