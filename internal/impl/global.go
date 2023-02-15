// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
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
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"go.jetpack.io/devbox/internal/ux"
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
	d.cfg.RawPackages = lo.Uniq(append(d.cfg.RawPackages, added...))
	if err := d.saveCfg(); err != nil {
		return err
	}
	return ensureGlobalProfileInPath()
}

func (d *Devbox) RemoveGlobal(pkgs ...string) error {
	profilePath, err := globalProfilePath()
	if err != nil {
		return err
	}
	if _, missing := lo.Difference(d.cfg.RawPackages, pkgs); len(missing) > 0 {
		ux.Fwarning(
			d.writer,
			"the following packages were not found in your global devbox.json: %s\n",
			strings.Join(missing, ", "),
		)
	}
	var removed []string
	for _, pkg := range lo.Intersect(d.cfg.RawPackages, pkgs) {
		if err := nix.ProfileRemove(profilePath, plansdk.DefaultNixpkgsCommit, pkg); err != nil {
			fmt.Fprintf(d.writer, "Error removing %s: %s", pkg, err)
		} else {
			fmt.Fprintf(d.writer, "%s was removed\n", pkg)
			removed = append(removed, pkg)
		}
	}
	d.cfg.RawPackages, _ = lo.Difference(d.cfg.RawPackages, removed)
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
	for _, p := range d.cfg.RawPackages {
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

	diff, _ := lo.Difference(pullCfg.RawPackages, d.cfg.RawPackages)
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

func GlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return filepath.Join(home, "/.config/devbox/"), nil
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

// Checks if the global profile is in the path
func ensureGlobalProfileInPath() error {
	profilePath, err := globalProfilePath()
	if err != nil {
		return err
	}
	binPath := filepath.Join(profilePath, "bin")
	if !strings.Contains(os.Getenv("PATH"), binPath) {
		return usererr.NewWarning(
			"devbox global profile is not in your PATH. Add `export PATH=$PATH:%s` to your shell config to fix this.", binPath,
		)
	}
	return nil
}
