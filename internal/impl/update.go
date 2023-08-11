// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"context"
	"fmt"

	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/nix/nixprofile"
	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/devbox/internal/shellgen"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/wrapnix"
)

func (d *Devbox) Update(ctx context.Context, pkgs ...string) error {
	inputs, err := d.inputsToUpdate(pkgs...)
	if err != nil {
		return err
	}

	pendingPackagesToUpdate := []*devpkg.Package{}
	for _, pkg := range inputs {
		if pkg.IsLegacy() {
			fmt.Fprintf(d.writer, "Updating %s -> %s\n", pkg.Raw, pkg.LegacyToVersioned())
			if err := d.Remove(ctx, pkg.Raw); err != nil {
				return err
			}
			// Calling Add function with the original package names, since
			// Add will automatically append @latest if search is able to handle that.
			// If not, it will fallback to the nixpkg format.
			if err := d.Add(ctx, pkg.Raw); err != nil {
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
			if err = d.updateDevboxPackage(ctx, pkg); err != nil {
				return err
			}
		}
	}

	if err := d.ensurePackagesAreInstalled(ctx, ensure); err != nil {
		return err
	}

	if err := wrapnix.CreateWrappers(ctx, d); err != nil {
		return err
	}

	return nix.FlakeUpdate(shellgen.FlakePath(d))
}

func (d *Devbox) inputsToUpdate(pkgs ...string) ([]*devpkg.Package, error) {
	var pkgsToUpdate []string
	for _, pkg := range pkgs {
		found, err := d.findPackageByName(pkg)
		if err != nil {
			return nil, err
		}
		pkgsToUpdate = append(pkgsToUpdate, found)
	}
	if len(pkgsToUpdate) == 0 {
		pkgsToUpdate = d.PackageNames()
	}

	return devpkg.PackageFromStrings(pkgsToUpdate, d.lockfile), nil
}

func (d *Devbox) updateDevboxPackage(
	ctx context.Context,
	pkg *devpkg.Package,
) error {
	resolved, err := d.lockfile.FetchResolvedPackage(pkg.Raw)
	if err != nil {
		return err
	}

	return d.mergeResolvedPackageToLockfile(ctx, pkg, resolved, d.lockfile)
}

func (d *Devbox) mergeResolvedPackageToLockfile(
	ctx context.Context,
	pkg *devpkg.Package,
	resolved *lock.Package,
	lockfile *lock.File,
) error {
	existing := lockfile.Packages[pkg.Raw]
	if existing == nil {
		ux.Finfo(d.writer, "Resolved %s to %[1]s %[2]s\n", pkg, resolved.Resolved)
		lockfile.Packages[pkg.Raw] = resolved
		return nil
	}

	if existing.Version != resolved.Version {
		ux.Finfo(d.writer, "Updating %s %s -> %s\n", pkg, existing.Version, resolved.Version)
		if err := d.removePackagesFromProfile(ctx, []string{pkg.Raw}); err != nil {
			// Warn but continue. TODO(landau): ensurePackagesAreInstalled should
			// sync the profile so we don't need to do this manually.
			ux.Fwarning(d.writer, "Failed to remove %s from profile: %s\n", pkg, err)
		}
		lockfile.Packages[pkg.Raw] = resolved
		return nil
	}

	// Add any missing system infos for packages whose versions did not change.
	if featureflag.RemoveNixpkgs.Enabled() {
		userSystem, err := nix.System()
		if err != nil {
			return err
		}

		if lockfile.Packages[pkg.Raw].Systems == nil {
			lockfile.Packages[pkg.Raw].Systems = map[string]*lock.SystemInfo{}
		}

		updated := false
		for sysName, newSysInfo := range resolved.Systems {
			if sysName == userSystem {
				// The resolved pkg has a system info for the user's system, so add/overwrite it.
				if !newSysInfo.Equals(existing.Systems[userSystem]) {
					// We only guard this so that the ux messaging is accurate. We could overwrite every time.
					lockfile.Packages[pkg.Raw].Systems[userSystem] = newSysInfo
					updated = true
				}
			} else {
				// Add other system infos if they don't exist, or if we have a different StorePath. This may
				// overwrite an existing CAPath, but to ensure correctness we should ensure that all StorePaths
				// come from the same package version.
				existingSysInfo, exists := existing.Systems[sysName]
				if !exists || existingSysInfo.StorePath != newSysInfo.StorePath {
					lockfile.Packages[pkg.Raw].Systems[sysName] = newSysInfo
					updated = true
				}
			}
		}
		if updated {
			ux.Finfo(d.writer, "Updated system information for %s\n", pkg)
			return nil
		}
	}

	ux.Finfo(d.writer, "Already up-to-date %s %s\n", pkg, existing.Version)
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
		d.writer,
		"Attempting to upgrade %s using `nix profile upgrade`\n",
		pkg.Raw,
	)

	err = nixprofile.ProfileUpgrade(profilePath, pkg, d.lockfile)
	if err != nil {
		ux.Ferror(
			d.writer,
			"Failed to upgrade %s using `nix profile upgrade`: %s\n",
			pkg.Raw,
			err,
		)
	}

	return nil
}
