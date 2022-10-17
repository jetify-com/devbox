// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plansdk

import (
	"encoding/json"
	"fmt"
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

// Plan tells devbox how to start shell projects.
type ShellPlan struct {
	// Set by devbox.json
	DevPackages []string `cue:"[...string]" json:"dev_packages,omitempty"`
	// Init hook on shell start. Currently, Nginx and python pip planners need it for shell.
	ShellInitHook []string `cue:"[...string]" json:"shell_init_hook,omitempty"`
	// Nix overlays. Currently, Rust needs it for shell.
	NixOverlays []string `cue:"[...string]" json:"nix_overlays,omitempty"`
	// Nix expressions. Currently, PHP needs it for shell.
	Definitions []string `cue:"[...string]" json:"definitions,omitempty"`
	// GeneratedFiles is a map of name => content for files that should be generated
	// in the .devbox/gen directory. (Use string to make it marshalled version nicer.)
	GeneratedFiles map[string]string `json:"generated_files,omitempty"`
}

// Plan tells devbox how to start build projects.
type BuildPlan struct {
	NixOverlays []string `cue:"[...string]" json:"nix_overlays,omitempty"`
	// DevPackages is the slice of Nix packages that devbox makes available in
	// its development environment. They are also available in shell.
	DevPackages []string `cue:"[...string]" json:"dev_packages"`

	// RuntimePackages is the slice of Nix packages that devbox makes available in
	// both the development environment and the final container that runs the
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

	Definitions []string `cue:"[...string]" json:"definitions,omitempty"`

	// Errors from plan generation. This usually means
	// the user application may not be buildable.
	Errors []PlanError `json:"errors,omitempty"`
}

type Planner interface {
	Name() string
	IsRelevant(srcDir string) bool
	GetBuildPlan(srcDir string) *BuildPlan
	GetShellPlan(srcDir string) *ShellPlan
}

// MergeShellPlans merges multiple Plans into one. The merged plan's packages, definitions,
// and overlays is the union of the packages, definitions, and overlays of the input plans,
// respectively.
func MergeShellPlans(plans ...*ShellPlan) (*ShellPlan, error) {
	shellPlan := &ShellPlan{}
	for _, p := range plans {
		err := mergo.Merge(shellPlan, p, mergo.WithAppendSlice)
		if err != nil {
			return nil, err
		}
	}

	shellPlan.DevPackages = pkgslice.Unique(shellPlan.DevPackages)
	shellPlan.Definitions = pkgslice.Unique(shellPlan.Definitions)
	shellPlan.NixOverlays = pkgslice.Unique(shellPlan.NixOverlays)
	shellPlan.ShellInitHook = pkgslice.Unique(shellPlan.ShellInitHook)

	return shellPlan, nil
}

func (p *BuildPlan) String() string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Invalid returns true if build plan has errors.
func (p *BuildPlan) Invalid() bool {
	return len(p.Errors) > 0
}

// Error combines all errors into a single error. We use this instead of a
// Error() string interface because some of the errors may be user errors, which
// get formatted differently by some clients.
func (p *BuildPlan) Error() error {
	if len(p.Errors) == 0 {
		return nil
	}
	combined := p.Errors[0].error
	for _, err := range p.Errors[1:] {
		combined = errors.Wrap(combined, err.Error())
	}
	return combined
}

func (p *BuildPlan) WithError(err error) *BuildPlan {
	p.Errors = append(p.Errors, PlanError{err})
	return p
}

// Get warning as error format from all 3 stages
func (p *BuildPlan) Warning() error {
	stages := []*Stage{p.InstallStage, p.BuildStage, p.StartStage}
	stageWarnings := []error{}
	for _, stage := range stages {
		if stage != nil && stage.Warning != nil {
			stageWarnings = append(stageWarnings, stage.Warning)
		}
	}
	if len(stageWarnings) == 0 {
		return nil
	}
	combined := stageWarnings[0]
	for _, err := range stageWarnings[1:] {
		combined = errors.Wrap(combined, err.Error())
	}
	return combined
}

func MergeUserBuildPlan(userPlan *BuildPlan, automatedPlan *BuildPlan) (*BuildPlan, error) {
	automatedStages := []*Stage{automatedPlan.InstallStage, automatedPlan.BuildStage, automatedPlan.StartStage}
	userStages := []*Stage{userPlan.InstallStage, userPlan.BuildStage, userPlan.StartStage}
	planStages := []*Stage{{}, {}, {}}

	for i := range planStages {
		planStages[i] = mergeUserStage(userStages[i], automatedStages[i])
	}
	// Merging devPackages and runtimePackages fields.
	packagesPlan := &BuildPlan{
		DevPackages:     append([]string{}, userPlan.DevPackages...),
		RuntimePackages: append([]string{}, userPlan.DevPackages...),
	}
	err := mergo.Merge(packagesPlan, automatedPlan, mergo.WithAppendSlice)
	if err != nil {
		return nil, err
	}
	plan := automatedPlan
	plan.InstallStage = planStages[0]
	plan.BuildStage = planStages[1]
	plan.StartStage = planStages[2]
	plan.DevPackages = pkgslice.Unique(packagesPlan.DevPackages)
	plan.RuntimePackages = pkgslice.Unique(packagesPlan.RuntimePackages)

	return plan, nil
}

func mergeUserStage(userStage *Stage, automatedStage *Stage) *Stage {
	stage := automatedStage
	if stage == nil {
		stage = &Stage{}
	}
	// Override commands
	if userStage != nil && userStage.Command != "" {
		stage.Command = userStage.Command
		// Clear out error as the user has overwritten the default.
		stage.Warning = nil
	}
	return stage
}

func (p PlanError) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Error())
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func WelcomeMessage(s string) string {
	return fmt.Sprintf(`echo "%s";`, s)
}
