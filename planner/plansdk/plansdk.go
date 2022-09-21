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

// TODO: Plan currently has a bunch of fields that it should not export.
// Two reasons why we need this right now:
// 1/ So that individual planners can use the fields
// 2/ So that we print them out correctly in `devbox plan`
//
// (1) can be solved by using a WithOption pattern, (e.g. NewPlan(..., WithWelcomeMessage(...)))
// (2) can be solved by using a custom JSON marshaler.

// Plan tells devbox how to start shells and build projects.
type Plan struct {
	ShellWelcomeMessage string `json:"shell_welcome_message,omitempty"`

	NixOverlays []string `cur:"[...string]" json:"nix_overlays,omitempty"`

	// DevPackages is the slice of Nix packages that devbox makes available in
	// its development environment. They are also available in shell.
	DevPackages []string `cue:"[...string]" json:"dev_packages"`

	// RuntimePackages is the slice of Nix packages that devbox makes available in
	// in both the development environment and the final container that runs the
	// application.
	RuntimePackages []string `cue:"[...string]" json:"runtime_packages"`
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

	Definitions []string `cue:"[...string]" json:"definitions"`
	// Errors from plan generation. This usually means
	// the user application may not be buildable.
	Errors []PlanError `json:"errors,omitempty"`

	// GeneratedFiles is a map of name => content for files that should be generated
	// in the .devbox/gen directory. (Use string to make it marshalled version nicer.)
	GeneratedFiles map[string]string `json:"generated_files,omitempty"`
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
	mergedPlan := &Plan{}
	for _, p := range plans {
		err := mergo.Merge(
			mergedPlan,
			&Plan{
				NixOverlays:     p.NixOverlays,
				DevPackages:     p.DevPackages,
				RuntimePackages: p.RuntimePackages,
				Definitions:     p.Definitions,
			},
			// Only WithAppendSlice overlays, definitions, dev, and runtime packages fields.
			mergo.WithAppendSlice,
		)
		if err != nil {
			return nil, err
		}
	}

	plan := findBuildablePlan(plans...)
	plan.NixOverlays = pkgslice.Unique(mergedPlan.NixOverlays)
	plan.DevPackages = pkgslice.Unique(mergedPlan.DevPackages)
	plan.RuntimePackages = pkgslice.Unique(mergedPlan.RuntimePackages)
	plan.Definitions = mergedPlan.Definitions

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
	plan := &Plan{
		InstallStage: userPlan.InstallStage,
		BuildStage:   userPlan.BuildStage,
		StartStage:   userPlan.StartStage,
	}
	// fields in plan:
	//   if empty, will inherit the corresponding fields in the automatedPlan
	//   if set, will override corresponding automatedPlan fields
	if err := mergo.Merge(plan, automatedPlan); err != nil {
		return nil, err
	}

	// Merging devPackages and runtimePackages fields.
	packagesPlan, err := MergePlans(userPlan, automatedPlan)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	plan.DevPackages = packagesPlan.DevPackages
	plan.RuntimePackages = packagesPlan.RuntimePackages

	return plan, nil
}

func (p PlanError) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Error())
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
