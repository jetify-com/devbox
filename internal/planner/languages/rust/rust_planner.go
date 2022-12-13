// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package rust

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cuecfg"
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

func (p *Planner) GetBuildPlan(srcDir string) *plansdk.BuildPlan {
	plan, err := p.getBuildPlan(srcDir)
	if err != nil {
		if plan == nil {
			plan = &plansdk.BuildPlan{}
		}
		plan.WithError(err)
	}
	return plan
}

func (p *Planner) getBuildPlan(srcDir string) (*plansdk.BuildPlan, error) {

	manifest, err := p.cargoManifest(srcDir)
	if err != nil {
		return nil, err
	}
	rustupVersion, err := p.rustupVersion(manifest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	envSetup := p.envsetupCommands(rustupVersion)

	return &plansdk.BuildPlan{
		// 'gcc' added as a linker for libc (C toolchain)
		// 1. https://softwareengineering.stackexchange.com/a/332254
		// 2. https://stackoverflow.com/a/56166959
		DevPackages:     []string{"rustup", "gcc"},
		RuntimePackages: []string{"rustup", "gcc"},

		InstallStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    fmt.Sprintf("%s && cargo fetch", envSetup),
		},
		BuildStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    fmt.Sprintf("%s && cargo build --release --offline", envSetup),
		},
		StartStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    fmt.Sprintf("%s && cargo run --release --offline", envSetup),
		},
	}, nil
}

// Follows the Rustup convention where it needs to be either:
// 1. stable
// 2. "<version>", including the quotation marks. Example: "1.62.0"
//
// TODO: add support for beta, nightly, and [stable|beta|nightly]-<archive-date>
// <channel>       = stable|beta|nightly|<major.minor>|<major.minor.patch>
// Channel names can be optionally appended with an archive date, as in nightly-2014-12-18
// https://rust-lang.github.io/rustup/concepts/toolchains.html
func (p *Planner) rustupVersion(manifest *cargoManifest) (string, error) {
	if manifest.PackageField.RustVersion == "" {
		return "stable", nil
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

// envsetupCommands are bash commands that ensure the rustup toolchain is setup so
// it always works. We tradeoff robustness for performance in this implementation,
// which is a polite way of saying that it is slow.
func (p *Planner) envsetupCommands(rustupVersion string) string {

	// RUSTUP_HOME sets the root rustup folder, which is used for storing installed toolchains
	// and configuration options. CARGO_HOME contains cache files used by cargo.
	//
	// Note that you will need to ensure these environment variables are always set and
	// that CARGO_HOME/bin is in the $PATH environment variable when using the toolchain.
	// source: https://rust-lang.github.io/rustup/installation/index.html
	cargoHome := "./.devbox/rust/cargo"
	cargoSetup := fmt.Sprintf("mkdir -p %s && export CARGO_HOME=%s && export PATH=$PATH:$CARGO_HOME", cargoHome,
		cargoHome)

	rustupHome := "./.devbox/rust/rustup/"
	rustupSetup := fmt.Sprintf("mkdir -p %s && export RUSTUP_HOME=%s && rustup default %s", rustupHome, rustupHome,
		rustupVersion)
	envSetup := fmt.Sprintf("%s && %s", cargoSetup, rustupSetup)

	return envSetup
}
