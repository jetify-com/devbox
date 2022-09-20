// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package rust

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/planner/plansdk"
)

// Source and reference: https://github.com/oxalica/rust-overlay
const RustOxalicaOverlay = "https://github.com/oxalica/rust-overlay/archive/stable.tar.gz"

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

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	plan, err := p.getPlan(srcDir)
	if err != nil {
		if plan == nil {
			plan = &plansdk.Plan{}
		}
		plan.WithError(err)
	}
	return plan
}

func (p *Planner) getPlan(srcDir string) (*plansdk.Plan, error) {

	manifest, err := p.cargoManifest(srcDir)
	if err != nil {
		return nil, err
	}
	rustVersion, err := p.rustOxalicaVersion(manifest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rustPkgDev := fmt.Sprintf("rust-bin.stable.%s.default", rustVersion)

	return &plansdk.Plan{
		NixOverlays: []string{RustOxalicaOverlay},
		// 'gcc' added as a linker for libc (C toolchain)
		// 1. https://softwareengineering.stackexchange.com/a/332254
		// 2. https://stackoverflow.com/a/56166959
		DevPackages:     []string{rustPkgDev, "gcc"},
		RuntimePackages: []string{"glibc"},
		InstallStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    "cargo fetch",
		},
		BuildStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    "cargo build --release --offline",
		},
		StartStage: &plansdk.Stage{
			InputFiles: []string{fmt.Sprintf("target/release/%s", manifest.PackageField.Name)},
			Command:    fmt.Sprintf("./%s", manifest.PackageField.Name),
		},
	}, nil
}

// Follows the Oxalica convention where it needs to be either:
// 1. latest
// 2. "<version>", including the quotation marks. Example: "1.62.0"
//
// This result is spliced into (for example) "rust-bin.stable.<result>.default"
func (p *Planner) rustOxalicaVersion(manifest *cargoManifest) (string, error) {
	if manifest.PackageField.RustVersion == "" {
		return "latest", nil
	}

	rustVersion, err := plansdk.NewVersion(manifest.PackageField.RustVersion)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("\"%s\"", rustVersion.Exact()), nil
}

type cargoManifest struct {
	// NOTE: 'package' is a protected keyword in golang so we cannot name this field 'package'.
	PackageField struct {
		Name        string `toml:"name,omitempty"`
		RustVersion string `toml:"rust-version,omitempty"`
	} `toml:"package,omitempty"`
}

func (p *Planner) cargoManifest(srcDir string) (*cargoManifest, error) {
	manifest := &cargoManifest{}
	// Since this Planner has been deemed relevant, we expect a valid cargoTomlPath
	err := cuecfg.ParseFile(p.cargoTomlPath(srcDir), manifest)
	return manifest, errors.WithStack(err)
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
