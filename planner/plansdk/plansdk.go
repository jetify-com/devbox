// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plansdk

import (
	"encoding/json"
	"os"

	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/pkgslice"
)

type PlanError struct {
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

	Definitions []string `cue:"[...string]" json:"definitions"`

	Errors []PlanError `json:"errors,omitempty"`
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

func (s *Stage) GetCommand() string {
	if s == nil {
		return ""
	}
	return s.Command
}

func (s *Stage) GetInputFiles() []string {
	if s == nil {
		return []string{}
	}
	return s.InputFiles
}

type Planner interface {
	Name() string
	IsRelevant(srcDir string) bool
	GetPlan(srcDir string) *Plan
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
	combined := p.Errors[0].error
	for _, err := range p.Errors[1:] {
		combined = errors.Wrap(combined, err.Error())
	}
	return combined
}

func (p *Plan) WithError(err error) *Plan {
	p.Errors = append(p.Errors, PlanError{err})
	return p
}

func MergePlans(plans ...*Plan) (*Plan, error) {
	plan := &Plan{}
	for _, p := range plans {
		err := mergo.Merge(
			plan,
			&Plan{
				DevPackages:     p.DevPackages,
				RuntimePackages: p.RuntimePackages,
				Definitions:     p.Definitions,
			},
			// Only WithAppendSlice definitions, dev, and runtime packages field.
			mergo.WithAppendSlice,
		)
		if err != nil {
			return nil, err
		}
	}

	plan.DevPackages = pkgslice.Unique(plan.DevPackages)
	plan.RuntimePackages = pkgslice.Unique(plan.RuntimePackages)
	plan.SharedPlan = findBuildablePlan(plans...).SharedPlan

	return plan, nil
}

func findBuildablePlan(plans ...*Plan) *Plan {
	for _, p := range plans {
		// For now, pick the first buildable plan.
		if p.Buildable() {
			return p
		}
	}
	return &Plan{}
}

func MergeUserPlan(userPlan *Plan, automatedPlan *Plan) (*Plan, error) {
	plan, err := MergePlans(userPlan, automatedPlan)
	if err != nil {
		return nil, err
	}
	sharedPlan := &Plan{
		SharedPlan: userPlan.SharedPlan,
	}
	// fields in sharedPlan:
	//   if empty, will inherit the corresponding fields in the automatedPlan
	//   if set, will override corresponding automatedPlan fields
	if err := mergo.Merge(sharedPlan, automatedPlan); err != nil {
		return nil, err
	}

	plan.SharedPlan = sharedPlan.SharedPlan

	return plan, nil
}

func (p PlanError) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Error())
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
