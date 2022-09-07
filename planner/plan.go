// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"encoding/json"

	"github.com/imdario/mergo"
)

type Plan struct {
	SharedPlan

	// DevPackages is the slice of Nix packages that devbox makes available in
	// its development environment.
	DevPackages []string `cue:"[...string]" json:"dev_packages"`
	// RuntimePackages is the slice of Nix packages that devbox makes available in
	// in both the development environment and the final container that runs the
	// application.
	RuntimePackages []string `cue:"[...string]" json:"runtime_packages"`
}

// Note: The SharedPlan struct is exposed in `devbox.json` – be thoughful of how
// we evolve the schema, and make sure we keep backwards compatibility.
type SharedPlan struct {
	// InstallStage defines the actions that should be taken when
	// installing language-specific libraries.
	// Ex: pip install, yarn install, go get
	InstallStage *Stage `json:"install_stage,omitempty"`
	// BuildStage defines the actions that should be taken when
	// compiling the application binary.
	// Ex: go build -o app
	BuildStage *Stage `json:"build_stage,omitempty"`
	// StartStage defines the actions that should be taken when
	// starting (running) the application.
	// Ex: python main.py
	StartStage *Stage `json:"start_stage,omitempty"`
}

type Stage struct {
	Command string `cue:"string" json:"command"`
	Image   string `json:"-"`
}

func (p *Plan) String() string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func MergePlans(plans ...*Plan) *Plan {
	plan := &Plan{
		DevPackages:     []string{},
		RuntimePackages: []string{},
	}
	for _, p := range plans {
		err := mergo.Merge(plan, p, mergo.WithAppendSlice)
		if err != nil {
			panic(err) // TODO: propagate error.
		}
	}
	return plan
}
