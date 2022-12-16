// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package rust

import (
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/internal/planner/plansdk"
)

// `cargo new` generates a file with uppercase Cargo.toml
const cargoToml = "Cargo.toml"

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "rust.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	return p.cargoTomlPath(srcDir) != ""
}

func (p *Planner) GetShellPlan(_srcDir string) *plansdk.ShellPlan {
	return &plansdk.ShellPlan{
		DevPackages: []string{"rustup"},
	}
}

// Tries to find Cargo.toml or cargo.toml. Returns the path with srcDir if found
// and empty-string if not found.
//
// NOTE: `cargo build` succeeded with lowercase cargo.toml, but `cargo build --release`
// will insist on `Cargo.toml`. We are lenient and tolerate both, until the user
// tries `devbox build` which relies upon `cargo build --release` to complain about this.
func (p *Planner) cargoTomlPath(srcDir string) string {

	cargoTomlPath := filepath.Join(srcDir, cargoToml)
	if plansdk.FileExists(cargoTomlPath) {
		return cargoTomlPath
	}

	lowerCargoTomlPath := filepath.Join(srcDir, strings.ToLower(cargoToml))
	if plansdk.FileExists(lowerCargoTomlPath) {
		return lowerCargoTomlPath
	}
	return ""
}
