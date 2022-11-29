// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plansdk

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/imdario/mergo"
	"go.jetpack.io/devbox/boxcli/featureflag"
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
	NixpkgsInfo *NixpkgsInfo
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

type Planner interface {
	Name() string
	IsRelevant(srcDir string) bool
	GetShellPlan(srcDir string) *ShellPlan
}

type PlannerForPackages interface {
	Planner
	IsRelevantForPackages(packages []string) bool
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

// publicly visible so that json marshalling works
type NixpkgsInfo struct {
	URL string

	Sha256 string
}

// Commit hash as of 2022-08-16
// `git ls-remote https://github.com/nixos/nixpkgs nixos-unstable`
const DefaultNixpkgsCommit = "af9e00071d0971eb292fd5abef334e66eda3cb69"

func GetNixpkgsInfo(commitHash string) (*NixpkgsInfo, error) {

	// If the featureflag is OFF, then we fallback to the hardcoded commit
	// and ignore any value set in the devbox.json
	if !featureflag.Get(featureflag.NixpkgVersion).Enabled() {
		// sha256 from:
		// nix-prefetch-url --unpack  https://github.com/nixos/nixpkgs/archive/<commit-hash>.tar.gz
		return &NixpkgsInfo{
			URL:    fmt.Sprintf("https://github.com/nixos/nixpkgs/archive/%s.tar.gz", DefaultNixpkgsCommit),
			Sha256: "1mdwy0419m5i9ss6s5frbhgzgyccbwycxm5nal40c8486bai0hwy",
		}, nil
	}

	return &NixpkgsInfo{
		URL: fmt.Sprintf("https://github.com/nixos/nixpkgs/archive/%s.tar.gz", commitHash),
	}, nil
}
