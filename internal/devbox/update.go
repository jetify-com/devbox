// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/nix/nixprofile"
	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/devbox/internal/shellgen"
	"go.jetpack.io/devbox/internal/ux"
)

func (d *Devbox) Update(ctx context.Context, opts devopt.UpdateOpts) error {
	inputs, err := d.inputsToUpdate(opts)
	if err != nil {
		return err
	}

	pendingPackagesToUpdate := []*devpkg.Package{}
	for _, pkg := range inputs {
		if pkg.IsLegacy() {
			fmt.Fprintf(d.stderr, "Updating %s -> %s\n", pkg.Raw, pkg.LegacyToVersioned())

			// Get the package from the config to get the Platforms and ExcludedPlatforms later
			cfgPackage, ok := d.cfg.Root.GetPackage(pkg.Raw)
			if !ok {
				return fmt.Errorf("package %s not found in config", pkg.Raw)
			}

			if err := d.Remove(ctx, pkg.Raw); err != nil {
				return err
			}
			// Calling Add function with the original package names, since
			// Add will automatically append @latest if search is able to handle that.
			// If not, it will fallback to the nixpkg format.
			if err := d.Add(ctx, []string{pkg.Raw}, devopt.AddOpts{
				Platforms:        cfgPackage.Platforms,
				ExcludePlatforms: cfgPackage.ExcludedPlatforms,
			}); err != nil {
				return err
			}
		} else {
			pendingPackagesToUpdate = append(pendingPackagesToUpdate, pkg)
		}
	}

	for _, pkg := range pendingPackagesToUpdate {
		if _, _, isVersioned := searcher.ParseVersionedPackage(pkg.Raw); !isVersioned {
			if err = d.attemptToUpgradeFlake(pkg); err != nil {
				return err
			}
		} else {
			if err = d.updateDevboxPackage(pkg); err != nil {
				return err
			}
		}
	}

	if err := d.ensureStateIsUpToDate(ctx, update); err != nil {
		return err
	}

	// I'm not entirely sure this is even needed, so ignoring the error.
	// It's definitely not needed for non-flakes. (which is 99.9% of packages)
	// It will return an error if .devbox/gen/flake is missing
	// TODO: Remove this if it's not needed.
	_ = nix.FlakeUpdate(shellgen.FlakePath(d))
	return nil
}

func (d *Devbox) inputsToUpdate(
	opts devopt.UpdateOpts,
) ([]*devpkg.Package, error) {
	if len(opts.Pkgs) == 0 {
		return d.AllPackages(), nil
	}

	var pkgsToUpdate []*devpkg.Package
	for _, pkg := range opts.Pkgs {
		found, err := d.findPackageByName(pkg)
		if opts.IgnoreMissingPackages && errors.Is(err, searcher.ErrNotFound) {
			continue
		} else if err != nil {
			return nil, err
		}
		pkgsToUpdate = append(pkgsToUpdate, found)
	}
	return pkgsToUpdate, nil
}

func (d *Devbox) updateDevboxPackage(pkg *devpkg.Package) error {
	resolved, err := d.lockfile.FetchResolvedPackage(pkg.Raw)
	if err != nil {
		return err
	}
	if resolved == nil {
		return nil
	}

	return d.mergeResolvedPackageToLockfile(pkg, resolved, d.lockfile)
}

func (d *Devbox) mergeResolvedPackageToLockfile(
	pkg *devpkg.Package,
	resolved *lock.Package,
	lockfile *lock.File,
) error {
	existing := lockfile.Packages[pkg.Raw]
	if existing == nil {
		ux.Finfo(d.stderr, "Resolved %s to %[1]s %[2]s\n", pkg, resolved.Resolved)
		lockfile.Packages[pkg.Raw] = resolved
		return nil
	}

	if existing.Version != resolved.Version {
		if existing.LastModified > resolved.LastModified {
			ux.Fwarning(
				d.stderr,
				"Resolved version for %s has older last_modified time. Not updating\n",
				pkg,
			)
			return nil
		}
		ux.Finfo(d.stderr, "Updating %s %s -> %s\n", pkg, existing.Version, resolved.Version)
		useResolvedPackageInLockfile(lockfile, pkg, resolved, existing)
		return nil
	}

	// Add any missing system infos for packages whose versions did not change.
	if featureflag.RemoveNixpkgs.Enabled() {

		if lockfile.Packages[pkg.Raw].Systems == nil {
			lockfile.Packages[pkg.Raw].Systems = map[string]*lock.SystemInfo{}
		}

		userSystem := nix.System()
		updated := false
		for sysName, newSysInfo := range resolved.Systems {
			// Check whether we are actually updating any system info.
			if sysName == userSystem {
				// The resolved pkg has a system info for the user's system, so add/overwrite it.
				if !newSysInfo.Equals(existing.Systems[userSystem]) {
					// We only guard this so that the ux messaging is accurate. We could overwrite every time.
					updated = true
				}
			} else {
				// Add other system infos if they don't exist, or if we have a different StorePath. This may
				// overwrite an existing StorePath, but to ensure correctness we should ensure that all StorePaths
				// come from the same package version.
				existingSysInfo, exists := existing.Systems[sysName]
				if !exists || !existingSysInfo.Equals(newSysInfo) {
					updated = true
				}
			}
		}
		if updated {
			// if we are updating the system info, then we should also update the other fields
			useResolvedPackageInLockfile(lockfile, pkg, resolved, existing)

			ux.Finfo(d.stderr, "Updated system information for %s\n", pkg)
			return nil
		}
	}

	ux.Finfo(d.stderr, "Already up-to-date %s %s\n", pkg, existing.Version)
	return nil
}

// attemptToUpgradeFlake attempts to upgrade a flake using `nix profile upgrade`
// and prints an error if it fails, but does not propagate upgrade errors.
func (d *Devbox) attemptToUpgradeFlake(pkg *devpkg.Package) error {
	profilePath, err := d.profilePath()
	if err != nil {
		return err
	}

	ux.Finfo(
		d.stderr,
		"Attempting to upgrade %s using `nix profile upgrade`\n",
		pkg.Raw,
	)

	err = nixprofile.ProfileUpgrade(profilePath, pkg, d.lockfile)
	if err != nil {
		ux.Fwarning(
			d.stderr,
			"Failed to upgrade %s using `nix profile upgrade`: %s\n",
			pkg.Raw,
			err,
		)
	}

	return nil
}

func useResolvedPackageInLockfile(
	lockfile *lock.File,
	pkg *devpkg.Package,
	resolved *lock.Package,
	existing *lock.Package,
) {
	lockfile.Packages[pkg.Raw] = resolved
	lockfile.Packages[pkg.Raw].AllowInsecure = existing.AllowInsecure
}
