// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/trace"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/devpkg/pkgtype"
	"go.jetpack.io/devbox/internal/shellgen"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/plugin"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/wrapnix"
)

// packages.go has functions for adding, removing and getting info about nix
// packages

// Add adds the `pkgs` to the config (i.e. devbox.json) and nix profile for this
// devbox project
// nolint:revive // warns about cognitive complexity
func (d *Devbox) Add(ctx context.Context, platforms, excludePlatforms []string, pkgsNames ...string) error {
	ctx, task := trace.NewTask(ctx, "devboxAdd")
	defer task.End()

	// Track which packages had no changes so we can report that to the user.
	unchangedPackageNames := []string{}

	// Only add packages that are not already in config. If same canonical exists,
	// replace it.
	pkgs := devpkg.PackageFromStrings(lo.Uniq(pkgsNames), d.lockfile)

	// addedPackageNames keeps track of the possibly transformed (versioned)
	// names of added packages (even if they are already in config). We use this
	// to know the exact name to mark as allowed insecure later on.
	addedPackageNames := []string{}
	existingPackageNames := d.PackageNames()
	for _, pkg := range pkgs {
		// If exact versioned package is already in the config, we can skip the
		// next loop that only deals with newPackages.
		if slices.Contains(existingPackageNames, pkg.Versioned()) {
			// But we still need to add to addedPackageNames. See its comment.
			addedPackageNames = append(addedPackageNames, pkg.Versioned())
			unchangedPackageNames = append(unchangedPackageNames, pkg.Versioned())
			ux.Finfo(d.stderr, "Package %q already in devbox.json\n", pkg.Versioned())
			continue
		}

		// On the other hand, if there's a package with same canonical name, replace
		// it. Ignore error (which is either missing or more than one). We search by
		// CanonicalName so any legacy or versioned packages will be removed if they
		// match.
		found, _ := d.findPackageByName(pkg.CanonicalName())
		if found != nil {
			ux.Finfo(d.stderr, "Replacing package %q in devbox.json\n", found.Raw)
			if err := d.Remove(ctx, found.Raw); err != nil {
				return err
			}
		}

		// validate that the versioned package exists in the search endpoint.
		// if not, fallback to legacy vanilla nix.
		versionedPkg := devpkg.PackageFromString(pkg.Versioned(), d.lockfile)

		packageNameForConfig := pkg.Raw
		ok, err := versionedPkg.ValidateExists(ctx)
		if (err == nil && ok) || errors.Is(err, devpkg.ErrCannotBuildPackageOnSystem) {
			// Only use versioned if it exists in search. We can disregard the error
			// about not building on the current system, since user's can continue
			// via --exclude-platform flag.
			packageNameForConfig = pkg.Versioned()
		} else if !versionedPkg.IsDevboxPackage() {
			// This means it didn't validate and we don't want to fallback to legacy
			// Just propagate the error.
			return err
		} else if _, err := nix.Search(d.lockfile.LegacyNixpkgsPath(pkg.Raw)); err != nil {
			// This means it looked like a devbox package or attribute path, but we
			// could not find it in search or in the legacy nixpkgs path.
			return usererr.New("Package %s not found", pkg.Raw)
		}

		ux.Finfo(d.stderr, "Adding package %q to devbox.json\n", packageNameForConfig)
		d.cfg.Packages.Add(packageNameForConfig)
		addedPackageNames = append(addedPackageNames, packageNameForConfig)
	}

	for _, pkg := range addedPackageNames {
		if err := d.cfg.Packages.AddPlatforms(d.stderr, pkg, platforms); err != nil {
			return err
		}
		if err := d.cfg.Packages.ExcludePlatforms(d.stderr, pkg, excludePlatforms); err != nil {
			return err
		}
	}

	// Resolving here ensures we allow insecure before running ensurePackagesAreInstalled
	// which will call print-dev-env. Resolving does not save the lockfile, we
	// save at the end when everything has succeeded.
	if d.allowInsecureAdds {
		for _, name := range addedPackageNames {
			p, err := d.lockfile.Resolve(name)
			if err != nil {
				return err
			}
			// TODO: Now that config packages can have fields,
			// we should set this in the config, not the lockfile.
			if !p.AllowInsecure {
				fmt.Fprintf(d.stderr, "Allowing insecure for %s\n", name)
			}
			p.AllowInsecure = true
		}
	}

	if err := d.ensurePackagesAreInstalled(ctx, install); err != nil {
		return usererr.WithUserMessage(err, "There was an error installing nix packages")
	}

	if err := d.saveCfg(); err != nil {
		return err
	}

	for _, input := range pkgs {
		if readme, err := plugin.Readme(
			ctx,
			input,
			d.projectDir,
			false /*markdown*/); err != nil {
			return err
		} else if readme != "" {
			fmt.Fprintf(d.stderr, "%s\n", readme)
		}
	}

	if len(platforms) == 0 && len(excludePlatforms) == 0 && !d.allowInsecureAdds {
		if len(unchangedPackageNames) == 1 {
			ux.Finfo(d.stderr, "Package %q was already in devbox.json and was not modified\n", unchangedPackageNames[0])
		} else if len(unchangedPackageNames) > 1 {
			ux.Finfo(d.stderr, "Packages %s were already in devbox.json and were not modified\n",
				strings.Join(unchangedPackageNames, ", "),
			)
		}
	}

	return nil
}

// Remove removes the `pkgs` from the config (i.e. devbox.json) and nix profile
// for this devbox project
func (d *Devbox) Remove(ctx context.Context, pkgs ...string) error {
	ctx, task := trace.NewTask(ctx, "devboxRemove")
	defer task.End()

	packagesToUninstall := []string{}
	missingPkgs := []string{}
	for _, pkg := range lo.Uniq(pkgs) {
		found, _ := d.findPackageByName(pkg)
		if found != nil {
			packagesToUninstall = append(packagesToUninstall, found.Raw)
			d.cfg.Packages.Remove(found.Raw)
		} else {
			missingPkgs = append(missingPkgs, pkg)
		}
	}

	if len(missingPkgs) > 0 {
		ux.Fwarning(
			d.stderr,
			"the following packages were not found in your devbox.json: %s\n",
			strings.Join(missingPkgs, ", "),
		)
	}

	if err := plugin.Remove(d.projectDir, packagesToUninstall); err != nil {
		return err
	}

	// this will clean up the now-extra package from nix profile and the lockfile
	if err := d.ensurePackagesAreInstalled(ctx, uninstall); err != nil {
		return err
	}

	return d.saveCfg()
}

// installMode is an enum for helping with ensurePackagesAreInstalled implementation
type installMode string

const (
	install   installMode = "install"
	uninstall installMode = "uninstall"
	// update is both install new package version and uninstall old package version
	update installMode = "update"
	ensure installMode = "ensure"
)

// ensurePackagesAreInstalled ensures:
//  1. Packages are installed, in nix-profile or runx.
//     Extraneous packages are removed (references purged, not uninstalled).
//  2. Files for devbox shellenv are generated
//  3. Env-vars for shellenv are computed
//  4. Lockfile is synced
//
// The `mode` is used for:
// 1. Skipping certain operations that may not apply.
// 2. User messaging to explain what operations are happening, because this function may take time to execute.
// TODO: Rename method since it does more than just ensure packages are installed.
func (d *Devbox) ensurePackagesAreInstalled(ctx context.Context, mode installMode) error {
	defer trace.StartRegion(ctx, "ensurePackages").End()
	defer debug.FunctionTimer().End()

	// if mode is install or uninstall, then we need to update the nix-profile
	// and lockfile, so we must continue below.
	if mode == ensure {
		// if mode is ensure, then we only continue if needed.
		if upToDate, err := d.lockfile.IsUpToDateAndInstalled(); err != nil || upToDate {
			return err
		}
		fmt.Fprintln(d.stderr, "Ensuring packages are installed.")
	}

	// Create plugin directories first because packages might need them
	for _, pkg := range d.InstallablePackages() {
		if err := d.PluginManager().Create(pkg); err != nil {
			return err
		}
	}

	if err := d.InstallRunXPackages(ctx); err != nil {
		return err
	}

	if err := shellgen.GenerateForPrintEnv(ctx, d); err != nil {
		return err
	}

	if err := plugin.RemoveInvalidSymlinks(d.projectDir); err != nil {
		return err
	}

	// Use the printDevEnvCache if we are adding or removing or updating any package,
	// AND we are not in the shellenv-enabled environment of the current devbox-project.
	usePrintDevEnvCache := mode != ensure && !d.IsEnvEnabled()
	nixEnv, err := d.computeNixEnv(ctx, usePrintDevEnvCache)
	if err != nil {
		return err
	}

	profile, err := d.profilePath()
	if err != nil {
		return err
	}
	if err := syncFlakeToProfile(ctx, d.flakeDir(), profile); err != nil {
		return err
	}

	// Ensure we clean out packages that are no longer needed.
	d.lockfile.Tidy()

	nixBins, err := d.nixBins(nixEnv)
	if err != nil {
		return err
	}

	if err := wrapnix.CreateWrappers(ctx, wrapnix.CreateWrappersArgs{
		NixBins:         nixBins,
		ProjectDir:      d.projectDir,
		ShellEnvHash:    nixEnv[d.shellEnvHashKey()],
		ShellEnvHashKey: d.shellEnvHashKey(),
	}); err != nil {
		return err
	}

	// Update lockfile with new packages that are not to be installed
	for _, pkg := range d.configPackages() {
		if err := pkg.EnsureUninstallableIsInLockfile(); err != nil {
			return err
		}
	}

	return d.lockfile.Save()
}

func (d *Devbox) profilePath() (string, error) {
	absPath := filepath.Join(d.projectDir, nix.ProfilePath)

	if err := resetProfileDirForFlakes(absPath); err != nil {
		debug.Log("ERROR: resetProfileDirForFlakes error: %v\n", err)
	}

	return absPath, errors.WithStack(os.MkdirAll(filepath.Dir(absPath), 0o755))
}

var resetCheckDone = false

// resetProfileDirForFlakes ensures the profileDir directory is cleared of old
// state if the Flakes feature has been changed, from the previous execution of a devbox command.
func resetProfileDirForFlakes(profileDir string) (err error) {
	if resetCheckDone {
		return nil
	}
	defer func() {
		if err == nil {
			resetCheckDone = true
		}
	}()

	dir, err := filepath.EvalSymlinks(profileDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.WithStack(err)
	}

	// older nix profiles have a manifest.nix file present
	_, err = os.Stat(filepath.Join(dir, "manifest.nix"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(os.Remove(profileDir))
}

func (d *Devbox) InstallRunXPackages(ctx context.Context) error {
	for _, pkg := range lo.Filter(d.InstallablePackages(), devpkg.IsRunX) {
		lockedPkg, err := d.lockfile.Resolve(pkg.Raw)
		if err != nil {
			return err
		}
		if _, err := pkgtype.RunXClient().Install(
			ctx,
			lockedPkg.Resolved,
		); err != nil {
			return fmt.Errorf("error installing runx package %s: %w", pkg, err)
		}
	}
	return nil
}
