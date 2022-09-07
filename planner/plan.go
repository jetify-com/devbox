// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"encoding/json"

	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/pkgslice"
)

type planError struct {
	error
}

type Plan struct {
	SharedPlan

	// DevPackages is the slice of Nix packages that devbox makes available in
	// its development environment.
	DevPackages []string `cue:"[...string]" json:"dev_packages"`
	// RuntimePackages is the slice of Nix packages that devbox makes available in
	// in both the development environment and the final container that runs the
	// application.
	RuntimePackages []string `cue:"[...string]" json:"runtime_packages"`

	Errors []planError `json:"errors,omitempty"`
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
	// InputFiles is internal for planners only.
	InputFiles []string `cue:"[...string]" json:"input_files,omitempty"`
}

func (p *Plan) String() string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (p *Plan) Buildable() bool {
	if p == nil {
		return false
	}
	return p.InstallStage != nil || p.BuildStage != nil || p.StartStage != nil
}

// Invalid returns true if plan is empty and has errors. If the plan is a partial
// plan, then it is considered valid.
func (p *Plan) Invalid() bool {
	return len(p.DevPackages) == 0 &&
		len(p.RuntimePackages) == 0 &&
		p.InstallStage == nil &&
		p.BuildStage == nil &&
		p.StartStage == nil &&
		len(p.Errors) > 0
}

// Error combines all errors into a single error. We use this instead of a
// Error() string interface because some of the errors may be user errors, which
// get formatted differently by some clients.
func (p *Plan) Error() error {
	if len(p.Errors) == 0 {
		return nil
	}
	var err error = p.Errors[0]
	for _, err = range p.Errors[1:] {
		err = errors.Wrap(err, err.Error())
	}
	return err
}

func (p *Plan) WithError(err error) *Plan {
	p.Errors = append(p.Errors, planError{err})
	return p
}

func MergePlans(plans ...*Plan) *Plan {
	plan := &Plan{
		DevPackages:     []string{},
		RuntimePackages: []string{},
		SharedPlan: SharedPlan{
			InstallStage: &Stage{},
			BuildStage:   &Stage{},
			StartStage:   &Stage{},
		},
	}
	for _, p := range plans {
		err := mergo.Merge(plan, p, mergo.WithAppendSlice)
		if err != nil {
			panic(err) // TODO: propagate error.
		}
	}

	plan.DevPackages = pkgslice.Unique(plan.DevPackages)
	plan.RuntimePackages = pkgslice.Unique(plan.RuntimePackages)

	// Set default files for install stage to copy.
	if plan.SharedPlan.InstallStage.InputFiles == nil {
		plan.SharedPlan.InstallStage.InputFiles = []string{"."}
	}
	// Set default files for install stage to copy over from build step.
	if plan.SharedPlan.StartStage.InputFiles == nil {
		plan.SharedPlan.StartStage.InputFiles = []string{"."}
	}

	return plan
}

func (p planError) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Error())
}
