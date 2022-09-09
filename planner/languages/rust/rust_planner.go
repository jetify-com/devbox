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

// `cargo new` generates a file with uppercase Cargo.toml, so we default to this
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

	// NOTE: I'm not convinced we need these packages to be included in the long term.
	// The link indicates we need libiconv for macOS, but maybe not for linux. For now,
	// including them by default, but this could be likely optimized once we understand better.
	//
	// libiconv due to error:
	//     ld: library not found for -liconv. clang-11: error: linker command failed with exit code 1
	// https://discourse.nixos.org/t/nix-shell-rust-hello-world-ld-linkage-issue/17381/2
	//
	// gcc due to error:
	//     linker `cc` not found
	// https://github.com/NixOS/nixpkgs/issues/103642
	packages := []string{"rustup", "libiconv", "gcc"}

	rustupVersion, err := p.rustVersion(srcDir)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rustupDefaultCmd := fmt.Sprintf("rustup default %s", rustupVersion)

	return &plansdk.Plan{
		DevPackages:     packages,
		RuntimePackages: packages,
		Shell: plansdk.PlanShell{
			PreInitHook: rustupDefaultCmd,
		},
	}, nil
}

func (p *Planner) rustVersion(srcDir string) (string, error) {
	cfg, err := p.cargoManifest(srcDir)
	if err != nil {
		return "", err
	}
	if cfg.PackageField.RustVersion == "" {
		return "stable", nil
	}

	if rustVersion, err := plansdk.NewVersion(cfg.PackageField.RustVersion); err != nil {
		return "", err
	} else {
		return rustVersion.Exact(), nil
	}
}

type cargoManifest struct {
	// NOTE: 'package' is a protected keyword in golang so we cannot name this field 'package'.
	PackageField struct {
		RustVersion string `toml:"rust-version,omitempty"`
	} `toml:"package,omitempty"`
}

func (p *Planner) cargoManifest(srcDir string) (*cargoManifest, error) {
	cargoTomlPath := filepath.Join(srcDir, cargoToml)
	cfg := &cargoManifest{}
	err := cuecfg.ReadFile(cargoTomlPath, cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return cfg, nil
}

// Tries to find Cargo.toml or cargo.toml. Returns the path with srcDir if found
// and empty-string if not found.
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
