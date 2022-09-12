// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package devbox creates isolated development environments.
package devbox

import (
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/docker"
	"go.jetpack.io/devbox/nix"
	"go.jetpack.io/devbox/pkgslice"
	"go.jetpack.io/devbox/planner"
	"go.jetpack.io/devbox/planner/plansdk"
	"golang.org/x/exp/slices"
)

// configFilename is name of the JSON file that defines a devbox environment.
const configFilename = "devbox.json"

// InitConfig creates a default devbox config file if one doesn't already
// exist.
func InitConfig(dir string) (created bool, err error) {
	cfgPath := filepath.Join(dir, configFilename)
	return cuecfg.InitFile(cfgPath, &Config{})
}

// Devbox provides an isolated development environment that contains a set of
// Nix packages.
type Devbox struct {
	cfg    *Config
	srcDir string
}

// Open opens a devbox by reading the config file in dir.
func Open(dir string) (*Devbox, error) {
	cfgPath := filepath.Join(dir, configFilename)

	cfg, err := ReadConfig(cfgPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	box := &Devbox{
		cfg:    cfg,
		srcDir: dir,
	}
	return box, nil
}

// Add adds a Nix package to the config so that it's available in the devbox
// environment. It validates that the Nix package exists, but doesn't install
// it. Adding a duplicate package is a no-op.
func (d *Devbox) Add(pkgs ...string) error {
	// Check packages are valid before adding.
	for _, pkg := range pkgs {
		ok := nix.PkgExists(pkg)
		if !ok {
			return errors.Errorf("package %s not found", pkg)
		}
	}

	// Add to Packages only if it's not already there
	for _, pkg := range pkgs {
		if slices.Contains(d.cfg.Packages, pkg) {
			continue
		}
		d.cfg.Packages = append(d.cfg.Packages, pkg)
	}
	return d.saveCfg()
}

// Remove removes Nix packages from the config so that it no longer exists in
// the devbox environment.
func (d *Devbox) Remove(pkgs ...string) error {
	// Remove packages from config.
	d.cfg.Packages = pkgslice.Exclude(d.cfg.Packages, pkgs)
	return d.saveCfg()
}

// Build creates a Docker image containing a shell with the devbox environment.
func (d *Devbox) Build(opts ...docker.BuildOptions) error {
	if ok, err := planner.IsBuildable(d.srcDir); !ok {
		return err
	}
	defaultFlags := &docker.BuildFlags{
		Name:           "devbox",
		DockerfilePath: filepath.Join(d.srcDir, ".devbox/gen", "Dockerfile"),
	}
	opts = append([]docker.BuildOptions{docker.WithFlags(defaultFlags)}, opts...)

	err := d.Generate()
	if err != nil {
		return errors.WithStack(err)
	}
	return docker.Build(d.srcDir, opts...)
}

// Plan creates a plan of the actions that devbox will take to generate its
// environment.
func (d *Devbox) Plan() (*plansdk.Plan, error) {
	userPlan := &plansdk.Plan{
		DevPackages:     d.cfg.Packages,
		RuntimePackages: d.cfg.Packages,
		SharedPlan:      d.cfg.SharedPlan,
	}

	return plansdk.MergeUserPlan(userPlan, planner.GetPlan(d.srcDir))
}

// Generate creates the directory of Nix files and the Dockerfile that define
// the devbox environment.
func (d *Devbox) Generate() error {
	plan, err := d.Plan()
	if err != nil {
		return errors.WithStack(err)
	}
	if plan.Invalid() {
		return plan.Error()
	}
	return generate(d.srcDir, plan, append(shellFiles, buildFiles...))
}

// Shell generates the devbox environment and launches nix-shell as a child
// process.
func (d *Devbox) Shell() error {
	plan, err := d.Plan()
	if err != nil {
		return errors.WithStack(err)
	}
	if plan.Invalid() {
		return plan.Error()
	}
	err = generate(d.srcDir, plan, shellFiles)
	if err != nil {
		return errors.WithStack(err)
	}
	nixDir := filepath.Join(d.srcDir, ".devbox/gen/shell.nix")
	sh, err := nix.DetectShell()
	if err != nil {
		// Fall back to using a plain Nix shell.
		sh = &nix.Shell{}
	}
	sh.UserInitHook = d.cfg.Shell.InitHook
	return sh.Run(nixDir)
}

// saveCfg writes the config file to the devbox directory.
func (d *Devbox) saveCfg() error {
	cfgPath := filepath.Join(d.srcDir, configFilename)
	return cuecfg.WriteFile(cfgPath, d.cfg)
}
