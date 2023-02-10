// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

func (d *Devbox) AddGlobal(pkgs ...string) error {
	profilePath, err := globalProfilePath()
	if err != nil {
		return err
	}
	// validate all packages exist. Don't install anything if any are missing
	for _, pkg := range pkgs {
		if !nix.FlakesPkgExists(plansdk.DefaultNixpkgsCommit, pkg) {
			return nix.ErrPackageNotFound
		}
	}
	var added []string
	for _, pkg := range pkgs {
		if err := nix.ProfileInstall(profilePath, plansdk.DefaultNixpkgsCommit, pkg); err != nil {
			fmt.Fprintf(d.writer, "Error installing %s: %s", pkg, err)
		} else {
			fmt.Fprintf(d.writer, "%s is now installed\n", pkg)
			added = append(added, pkg)
		}
	}
	d.cfg.Packages = lo.Uniq(append(d.cfg.Packages, added...))
	return d.saveCfg()
}

func (d *Devbox) RemoveGlobal(pkgs ...string) error {
	profilePath, err := globalProfilePath()
	if err != nil {
		return err
	}
	if _, missing := lo.Difference(d.cfg.Packages, pkgs); len(missing) > 0 {
		fmt.Fprintf(
			d.writer,
			"%s the following packages were not found in your global devbox.json: %s\n",
			color.HiYellowString("Warning:"),
			strings.Join(missing, ", "),
		)
	}
	var removed []string
	for _, pkg := range lo.Intersect(d.cfg.Packages, pkgs) {
		if err := nix.ProfileRemove(profilePath, plansdk.DefaultNixpkgsCommit, pkg); err != nil {
			fmt.Fprintf(d.writer, "Error removing %s: %s", pkg, err)
		} else {
			fmt.Fprintf(d.writer, "%s was removed\n", pkg)
			removed = append(removed, pkg)
		}
	}
	d.cfg.Packages, _ = lo.Difference(d.cfg.Packages, removed)
	return d.saveCfg()
}

func (d *Devbox) PrintGlobalList() error {
	for _, p := range d.cfg.Packages {
		fmt.Fprintf(d.writer, "* %s\n", p)
	}
	return nil
}

func globalProfilePath() (string, error) {
	configPath, err := GlobalConfigPath()
	if err != nil {
		return "", err
	}
	nixDirPath := filepath.Join(configPath, "nix")
	_ = os.MkdirAll(nixDirPath, 0755)
	return filepath.Join(nixDirPath, "profile"), nil
}

func GlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return filepath.Join(home, "/.config/devbox/"), nil
}
