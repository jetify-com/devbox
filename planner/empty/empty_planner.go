// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Dummy empty planner. Mainly to serve as a template for how to start a new
// planner.

package empty

import "go.jetpack.io/devbox/planner/plansdk"

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "empty.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	return false
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	return &plansdk.Plan{}
}
