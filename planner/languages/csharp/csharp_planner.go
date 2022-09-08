// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package csharp

import (
	"go.jetpack.io/devbox/planner/plansdk"
)

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (g *Planner) Name() string {
	return "csharp.Planner"
}

func (g *Planner) IsRelevant(srcDir string) bool {
	return false
}

func (g *Planner) GetPlan(srcDir string) *plansdk.Plan {
	return &plansdk.Plan{}
}
