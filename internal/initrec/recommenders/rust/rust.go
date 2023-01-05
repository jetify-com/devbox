// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package rust

import (
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/internal/initrec/recommenders"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

// `cargo new` generates a file with uppercase Cargo.toml
const cargoToml = "Cargo.toml"

type Recommender struct {
	SrcDir string
}

// implements interface Recommender (compile-time check)
var _ recommenders.Recommender = (*Recommender)(nil)

func (r *Recommender) IsRelevant() bool {
	return cargoTomlPath(r.SrcDir) != ""
}

func (r *Recommender) Packages() []string {
	return []string{"rustup"}
}

// Tries to find Cargo.toml or cargo.toml. Returns the path with srcDir if found
// and empty-string if not found.
//
// NOTE: `cargo build` succeeded with lowercase cargo.toml, but `cargo build --release`
// will insist on `Cargo.toml`. We are lenient and tolerate both.
func cargoTomlPath(srcDir string) string {

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
