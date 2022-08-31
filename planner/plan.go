// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"encoding/json"

	"github.com/imdario/mergo"
)

// Note: The Plan struct is exposed in `devbox.json` – be thoughful of how
// we evolve the schema, and make sure we keep backwards compatibility.

type Plan struct {
	// Packages is the slice of Nix packages that devbox makes available in
	// its environment.
	Packages []string `cue:"[...string]" json:"packages"`
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
		Packages: []string{},
	}
	for _, p := range plans {
		err := mergo.Merge(plan, p, mergo.WithAppendSlice)
		if err != nil {
			panic(err) // TODO: propagate error.
		}
	}
	return plan
}
