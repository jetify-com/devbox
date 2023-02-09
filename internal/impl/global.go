// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"fmt"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

func (d *Devbox) AddGlobal(pkgs ...string) error {
	for _, pkg := range pkgs {
		if err := nix.ProfileInstall(plansdk.DefaultNixpkgsCommit, pkg); err != nil {
			// TODO: we should only add packages to devbox.json if we actually
			// installed them in the nix profile.
			return err
		}
	}
	d.cfg.Packages = lo.Uniq(append(d.cfg.Packages, pkgs...))
	return d.saveCfg()
}

func (d *Devbox) RemoveGlobal(pkgs ...string) error {
	for _, pkg := range pkgs {
		if err := nix.ProfileRemove(plansdk.DefaultNixpkgsCommit, pkg); err != nil {
			// TODO: we should only remove packages from devbox.json if we actually
			// removed them from the nix profile.
			return err
		}
	}
	d.cfg.Packages, _ = lo.Difference(d.cfg.Packages, pkgs)
	return d.saveCfg()
}

func (d *Devbox) PrintGlobalList() error {
	for _, p := range d.cfg.Packages {
		fmt.Fprintf(d.writer, "* %s\n", p)
	}
	return nil
}
