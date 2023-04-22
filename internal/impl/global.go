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
	"go.jetpack.io/devbox/internal/env"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/xdg"
)

var warningNotInPath = usererr.NewWarning(`the devbox global profile is not in your $PATH.

Add the following line to your shell's rcfile (e.g., ~/.bashrc or ~/.zshrc)
and restart your shell to fix this:

	eval "$(devbox global shellenv)"
`)

// In the future we will support multiple global profiles
const currentGlobalProfile = "default"

func (d *Devbox) AddGlobal(pkgs ...string) error {
	pkgs = lo.Uniq(pkgs)
	globalRoot, err := GlobalDataPath()
	if err != nil {
		return err
	}

	// validate all packages exist. Don't install anything if any are missing
	for _, pkg := range pkgs {
		found, err := nix.PkgExists(plansdk.DefaultNixpkgsCommit, pkg, globalRoot)
		if err != nil {
			return err
		} else if !found {
			return nix.ErrPackageNotFound
		}
	}
	var added []string
	profilePath, err := GlobalNixProfilePath()
	if err != nil {
		return err
	}

	total := len(pkgs)
	for idx, pkg := range pkgs {
		stepNum := idx + 1
		stepMsg := fmt.Sprintf("[%d/%d] %s", stepNum, total, pkg)
		err = nix.ProfileInstall(&nix.ProfileInstallArgs{
			CustomStepMessage: stepMsg,
			NixpkgsCommit:     d.cfg.Nixpkgs.Commit,
			Package:           pkg,
			ProfilePath:       profilePath,
			ProjectDir:        globalRoot,
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
	d.cfg.RawPackages = lo.Uniq(append(d.cfg.RawPackages, added...))
	if err := d.saveCfg(); err != nil {
		return err
	}
	return ensureGlobalProfileInPath()
}

func (d *Devbox) RemoveGlobal(pkgs ...string) error {
	pkgs = lo.Uniq(pkgs)
	if _, missing := lo.Difference(d.cfg.RawPackages, pkgs); len(missing) > 0 {
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
	for _, pkg := range lo.Intersect(d.cfg.RawPackages, pkgs) {
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

func globalBinPath() (string, error) {
	nixProfilePath, err := GlobalNixProfilePath()
	if err != nil {
		return "", err
	}
	currentPath := xdg.DataSubpath("devbox/global/current")
	// For now default is always current. In the future we will support multiple
	// and allow user to switch.
	if err := os.Symlink(nixProfilePath, currentPath); err != nil && !os.IsExist(err) {
		return "", errors.WithStack(err)
	}
	return filepath.Join(currentPath, "bin"), nil
}

// Checks if the global profile is in the path
func ensureGlobalProfileInPath() error {
	binPath, err := globalBinPath()
	if err != nil {
		return err
	}
	if !strings.Contains(os.Getenv(env.Path), binPath) {
		return warningNotInPath
	}
	return nil
}
