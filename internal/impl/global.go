// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

func (d *Devbox) AddGlobal(pkgs ...string) error {
	// validate all packages exist. Don't install anything if any are missing
	for _, pkg := range pkgs {
		if !nix.FlakesPkgExists(plansdk.DefaultNixpkgsCommit, pkg) {
			return nix.ErrPackageNotFound
		}
	}
	var added, installErrors []string
	for _, pkg := range pkgs {
		if err := nix.ProfileInstall(plansdk.DefaultNixpkgsCommit, pkg); err != nil {
			installErrors = append(installErrors, err.Error())
		} else {
			fmt.Fprintf(d.writer, "%s is now installed\n", pkg)
			added = append(added, pkg)
		}
	}
	d.cfg.Packages = lo.Uniq(append(d.cfg.Packages, added...))
	return errors.Wrap(d.saveCfg(), strings.Join(installErrors, " "))
}

func (d *Devbox) RemoveGlobal(pkgs ...string) error {
	if _, missing := lo.Difference(d.cfg.Packages, pkgs); len(missing) > 0 {
		fmt.Fprintf(
			d.writer,
			"%s the following packages were not found in your global devbox.json: %s\n",
			color.HiYellowString("Warning:"),
			strings.Join(missing, ", "),
		)
	}
	var removed, removeErrors []string
	for _, pkg := range lo.Intersect(d.cfg.Packages, pkgs) {
		if err := nix.ProfileRemove(plansdk.DefaultNixpkgsCommit, pkg); err != nil {
			removeErrors = append(removeErrors, err.Error())
		} else {
			fmt.Fprintf(d.writer, "%s was removed\n", pkg)
			removed = append(removed, pkg)
		}
	}
	d.cfg.Packages, _ = lo.Difference(d.cfg.Packages, removed)
	return errors.Wrap(d.saveCfg(), strings.Join(removeErrors, " "))
}

func (d *Devbox) PrintGlobalList() error {
	for _, p := range d.cfg.Packages {
		fmt.Fprintf(d.writer, "* %s\n", p)
	}
	return nil
}
