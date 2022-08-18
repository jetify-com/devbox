// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/docker"
	"go.jetpack.io/devbox/nix"
	"go.jetpack.io/devbox/planner"
)

type Devbox struct {
	cfg    *Config
	srcDir string
}

const CONFIG_FILENAME = "devbox.json"

func Init(dir string) (bool, error) {
	cfgPath := filepath.Join(dir, CONFIG_FILENAME)
	return cuecfg.InitFile(cfgPath, &Config{})
}

func Open(dir string) (*Devbox, error) {
	cfgPath := filepath.Join(dir, CONFIG_FILENAME)

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

func (d *Devbox) Add(pkgs ...string) error {
	// Check packages exist before adding.
	for _, pkg := range pkgs {
		ok := nix.PkgExists(pkg)
		if !ok {
			return errors.Errorf("Package %s not found.", pkg)
		}
	}
	// Merge and remove duplicates:
	merged := append(d.cfg.Packages, pkgs...)
	d.cfg.Packages = lo.FindUniques(merged)

	// Save config.
	return d.saveCfg()
}

func (d *Devbox) Remove(pkgs ...string) error {
	// Remove packages from config.
	d.cfg.Packages = lo.Without(d.cfg.Packages, pkgs...)

	// Save config.
	return d.saveCfg()
}

func (d *Devbox) Build(opts ...docker.BuildOptions) error {
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

func (d *Devbox) Plan() *planner.BuildPlan {
	basePlan := &planner.BuildPlan{
		Packages: d.cfg.Packages,
	}
	return planner.MergePlans(basePlan, planner.Plan(d.srcDir))
}

func (d *Devbox) Generate() error {
	plan := d.Plan()
	return generate(d.srcDir, plan)
}

func (d *Devbox) Shell() error {
	err := d.Generate()
	if err != nil {
		return errors.WithStack(err)
	}
	nixDir := filepath.Join(d.srcDir, ".devbox/gen")
	return nix.Shell(nixDir)
}

func (d *Devbox) saveCfg() error {
	cfgPath := filepath.Join(d.srcDir, CONFIG_FILENAME)
	return cuecfg.WriteFile(cfgPath, d.cfg)
}
