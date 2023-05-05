// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/xdg"
)

var warningNotInPath = `the devbox global profile is not in your $PATH.

Add the following line to your shell's rcfile (e.g., ~/.bashrc or ~/.zshrc)
and restart your shell to fix this:

	eval "$(devbox global shellenv)"
`

// In the future we will support multiple global profiles
const currentGlobalProfile = "default"

func (d *Devbox) AddGlobal(pkgs ...string) error {
	pkgs = lo.Uniq(pkgs)

	// validate all packages exist. Don't install anything if any are missing
	for _, pkg := range pkgs {
		found, err := nix.PkgExists(pkg, d.lockfile)
		if err != nil {
			return err
		}
		if !found {
			return nix.ErrPackageNotFound
		}
	}
	profilePath, err := GlobalNixProfilePath()
	if err != nil {
		return err
	}

	var added []string
	total := len(pkgs)
	for idx, pkg := range pkgs {
		stepNum := idx + 1
		stepMsg := fmt.Sprintf("[%d/%d] %s", stepNum, total, pkg)
		err = nix.ProfileInstall(&nix.ProfileInstallArgs{
			CustomStepMessage: stepMsg,
			Lockfile:          d.lockfile,
			Package:           pkg,
			ProfilePath:       profilePath,
			Writer:            d.writer,
		})
		if err != nil {
			fmt.Fprintf(d.writer, "Error installing %s: %s", pkg, err)
		} else {
			added = append(added, pkg)
		}
	}
	if len(added) == 0 && err != nil {
		return err
	}
	d.cfg.Packages = lo.Uniq(append(d.cfg.Packages, added...))
	if err := d.saveCfg(); err != nil {
		return err
	}
	d.ensureDevboxGlobalShellenvEnabled()
	return nil
}

func (d *Devbox) RemoveGlobal(pkgs ...string) error {
	pkgs = lo.Uniq(pkgs)
	if _, missing := lo.Difference(d.cfg.Packages, pkgs); len(missing) > 0 {
		ux.Fwarning(
			d.writer,
			"the following packages were not found in your global devbox.json: %s\n",
			strings.Join(missing, ", "),
		)
	}
	var removed []string
	profilePath, err := GlobalNixProfilePath()
	if err != nil {
		return err
	}
	for _, pkg := range lo.Intersect(d.cfg.Packages, pkgs) {
		if err := nix.ProfileRemove(profilePath, plansdk.DefaultNixpkgsCommit, pkg); err != nil {
			if errors.Is(err, nix.ErrPackageNotInstalled) {
				removed = append(removed, pkg)
			} else {
				fmt.Fprintf(d.writer, "Error removing %s: %s", pkg, err)
			}
		} else {
			fmt.Fprintf(d.writer, "%s was removed\n", pkg)
			removed = append(removed, pkg)
		}
	}
	d.cfg.Packages, _ = lo.Difference(d.cfg.Packages, removed)
	return d.saveCfg()
}

func (d *Devbox) PullGlobal(path string) error {
	u, err := url.Parse(path)
	if err == nil && u.Scheme != "" {
		return d.pullGlobalFromURL(u)
	}
	return d.pullGlobalFromPath(path)
}

func (d *Devbox) PrintGlobalList() error {
	for _, p := range d.cfg.Packages {
		fmt.Fprintf(d.writer, "* %s\n", p)
	}
	return nil
}

func (d *Devbox) pullGlobalFromURL(u *url.URL) error {
	fmt.Fprintf(d.writer, "Pulling global config from %s\n", u)
	cfg, err := readConfigFromURL(u)
	if err != nil {
		return err
	}
	return d.addFromPull(cfg)
}

func (d *Devbox) pullGlobalFromPath(path string) error {
	fmt.Fprintf(d.writer, "Pulling global config from %s\n", path)
	cfg, err := readConfig(path)
	if err != nil {
		return err
	}
	return d.addFromPull(cfg)
}

func (d *Devbox) addFromPull(pullCfg *Config) error {
	if pullCfg.Nixpkgs.Commit != plansdk.DefaultNixpkgsCommit {
		// TODO: For now show this warning, but we do plan to allow packages from
		// multiple commits in the future
		ux.Fwarning(d.writer, "nixpkgs commit mismatch. Using local one by default\n")
	}

	diff, _ := lo.Difference(pullCfg.Packages, d.cfg.Packages)
	if len(diff) == 0 {
		fmt.Fprint(d.writer, "No new packages to install\n")
		return nil
	}
	fmt.Fprintf(
		d.writer,
		"Installing the following packages: %s\n",
		strings.Join(diff, ", "),
	)
	return d.AddGlobal(diff...)
}

func GlobalDataPath() (string, error) {
	path := xdg.DataSubpath(filepath.Join("devbox/global", currentGlobalProfile))
	return path, errors.WithStack(os.MkdirAll(path, 0755))
}

func GlobalNixProfilePath() (string, error) {
	path, err := GlobalDataPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(path, "profile"), nil
}

// Checks if the global has been shellenv'd and warns the user if not
func (d *Devbox) ensureDevboxGlobalShellenvEnabled() {
	if os.Getenv(d.ogPathKey()) == "" {
		ux.Fwarning(d.writer, warningNotInPath)
	}
}
