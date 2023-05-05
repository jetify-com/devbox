// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plansdk

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/imdario/mergo"
	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/envir"
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

// ShellPlan tells devbox how to start shell projects.
type ShellPlan struct {
	NixpkgsInfo *NixpkgsInfo
	// Set by devbox.json
	DevPackages []string `json:"dev_packages,omitempty"`
	// Init hook on shell start. Currently, Nginx and python pip planners need it for shell.
	ShellInitHook []string `json:"shell_init_hook,omitempty"`
	// Nix expressions. Currently, PHP needs it for shell.
	Definitions map[string]string `json:"definitions,omitempty"`
	// GeneratedFiles is a map of name => content for files that should be generated
	// in the .devbox/gen directory. (Use string to make it marshalled version nicer.)
	GeneratedFiles map[string]string `json:"generated_files,omitempty"`

	FlakeInputs []*FlakeInput
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
// and shellInitHooks is the union of the packages, definitions, and shellInitHooks of the input plans,
// respectively.
func MergeShellPlans(plans ...*ShellPlan) (*ShellPlan, error) {
	shellPlan := &ShellPlan{}
	for _, p := range plans {
		err := mergo.Merge(shellPlan, p, mergo.WithAppendSlice)
		if err != nil {
			return nil, err
		}
	}

	shellPlan.DevPackages = lo.Uniq(shellPlan.DevPackages)
	shellPlan.ShellInitHook = lo.Uniq(shellPlan.ShellInitHook)

	return shellPlan, nil
}

func (p PlanError) MarshalJSON() ([]byte, error) {
	return cuecfg.MarshalJSON(p.Error())
}

func WelcomeMessage(s string) string {
	return fmt.Sprintf(`echo "%s";`, s)
}

// publicly visible so that json marshalling works
// TODO: We can probably get rid of this once we remove the haskell and php
// planners
type NixpkgsInfo struct {
	URL string
}

// The commit hash for nixpkgs-unstable on 2023-01-25 from status.nixos.org
const DefaultNixpkgsCommit = "f80ac848e3d6f0c12c52758c0f25c10c97ca3b62"

func GetNixpkgsInfo(commitHash string) *NixpkgsInfo {
	mirror := nixpkgsMirrorURL(commitHash)
	if mirror != "" {
		return &NixpkgsInfo{
			URL: mirror,
		}
	}
	return &NixpkgsInfo{
		URL: fmt.Sprintf("github:NixOS/nixpkgs/%s", commitHash),
	}
}

func nixpkgsMirrorURL(commitHash string) string {
	baseURL := os.Getenv(envir.DevboxCache)
	if baseURL == "" {
		return ""
	}

	// Check that the mirror is responsive and has the tar file. We can't
	// leave this up to Nix because fetchTarball will retry indefinitely.
	client := &http.Client{Timeout: 3 * time.Second}
	mirrorURL := fmt.Sprintf("%s/nixos/nixpkgs/archive/%s.tar.gz", baseURL, commitHash)
	resp, err := client.Head(mirrorURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	return mirrorURL
}
