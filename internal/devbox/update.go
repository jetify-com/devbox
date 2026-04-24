// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/devpkg"
	"go.jetify.com/devbox/internal/devpkg/pkgtype"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/internal/plugin"
	"go.jetify.com/devbox/internal/searcher"
	"go.jetify.com/devbox/internal/shellgen"
	"go.jetify.com/devbox/internal/ux"
	"go.jetify.com/devbox/nix/flake"
)

func (d *Devbox) Update(ctx context.Context, opts devopt.UpdateOpts) error {
	if len(opts.Pkgs) == 0 || slices.Contains(opts.Pkgs, "nixpkgs") {
		if err := d.lockfile.UpdateStdenv(); err != nil {
			return err
		}
		// if nixpkgs is the only package to update, just return here.
		if len(opts.Pkgs) == 1 {
			return nil
		}
		// Otherwise, remove nixpkgs and continue
		opts.Pkgs = slices.DeleteFunc(opts.Pkgs, func(pkg string) bool {
			return pkg == "nixpkgs"
		})
	}

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
		if pkgtype.IsFlake(pkg.Raw) {
			// Flake refs are updated by re-resolving the ref via `nix flake
			// metadata` and rewriting the lockfile entry. Errors are non-fatal
			// (network blip, deleted branch, renamed attr) — warn and continue
			// so one broken ref doesn't abort update for everything else.
			if err := d.updateDevboxPackage(pkg); err != nil {
				ux.Fwarningf(d.stderr, "Failed to update %s: %s\n", pkg.Raw, err)
			}
			continue
		}
		if _, _, isVersioned := searcher.ParseVersionedPackage(pkg.Raw); isVersioned {
			if err = d.updateDevboxPackage(pkg); err != nil {
				return err
			}
		}
	}

	d.packagesBeingUpdated = inputs

	mode := update
	if opts.NoInstall {
		mode = noInstall
	}
	if err := d.ensureStateIsUpToDate(ctx, mode); err != nil {
		return err
	}

	// I'm not entirely sure this is even needed, so ignoring the error.
	// It's definitely not needed for non-flakes. (which is 99.9% of packages)
	// It will return an error if .devbox/gen/flake is missing
	// TODO: Remove this if it's not needed.
	_ = nix.FlakeUpdate(shellgen.FlakePath(d))

	// fix any missing store paths.
	if err = d.FixMissingStorePaths(ctx); err != nil {
		return errors.WithStack(err)
	}

	return plugin.Update()
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
	// refresh=true so flake refs bypass nix's own metadata cache and re-query
	// upstream. Without this, `devbox update` on a github: ref can return a
	// stale commit that nix had cached from an earlier call.
	resolved, err := d.lockfile.FetchResolvedPackage(pkg.Raw, true)
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
		ux.Finfof(d.stderr, "Resolved %s to %[1]s %[2]s\n", pkg, resolved.Resolved)
		lockfile.Packages[pkg.Raw] = resolved
		return nil
	}

	// Flake refs have no Version, so the Version-based comparison below would
	// always report "Already up-to-date" even when the locked rev changed.
	// Handle them via their Resolved field (which embeds the locked rev) and
	// LastModified.
	if pkgtype.IsFlake(pkg.Raw) {
		return d.mergeResolvedFlakeToLockfile(pkg, resolved, existing, lockfile)
	}

	if existing.Version != resolved.Version {
		if existing.LastModified > resolved.LastModified {
			ux.Fwarningf(
				d.stderr,
				"Resolved version for %s has older last_modified time. Not updating\n",
				pkg,
			)
			return nil
		}
		ux.Finfof(d.stderr, "Updating %s %s -> %s\n", pkg, existing.Version, resolved.Version)
		useResolvedPackageInLockfile(lockfile, pkg, resolved, existing)
		return nil
	}

	// Add any missing system infos for packages whose versions did not change.
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

		ux.Finfof(d.stderr, "Updated system information for %s\n", pkg)
		return nil
	}

	ux.Finfof(d.stderr, "Already up-to-date %s %s\n", pkg, existing.Version)
	return nil
}

// mergeResolvedFlakeToLockfile updates the lockfile entry for a flake ref. It
// compares on Resolved (which embeds the locked rev) rather than Version since
// flake refs don't carry a semver. It honors the same LastModified staleness
// guard as the nixpkgs path.
func (d *Devbox) mergeResolvedFlakeToLockfile(
	pkg *devpkg.Package,
	resolved *lock.Package,
	existing *lock.Package,
	lockfile *lock.File,
) error {
	if existing.Resolved == resolved.Resolved {
		ux.Finfof(d.stderr, "Already up-to-date %s\n", pkg)
		return nil
	}

	// RFC3339 sorts lexicographically the same as chronologically; matches the
	// nixpkgs branch's comparison style a few lines above.
	if existing.LastModified > resolved.LastModified {
		ux.Fwarningf(
			d.stderr,
			"Resolved ref for %s has older last_modified time. Not updating\n",
			pkg,
		)
		return nil
	}

	ux.Finfof(d.stderr, "Updating %s %s\n", pkg, describeFlakeUpdate(existing, resolved))
	useResolvedPackageInLockfile(lockfile, pkg, resolved, existing)
	return nil
}

// describeFlakeUpdate renders a short human-readable diff between two flake
// lockfile entries. It prefers short revs when both sides have them, falls
// back to a date range when not, and omits either piece cleanly if missing.
func describeFlakeUpdate(existing, resolved *lock.Package) string {
	var parts []string
	if oldRev, newRev := shortRev(existing.Resolved), shortRev(resolved.Resolved); oldRev != "" && newRev != "" {
		parts = append(parts, fmt.Sprintf("%s -> %s", oldRev, newRev))
	}
	if oldDate, newDate := shortDate(existing.LastModified), shortDate(resolved.LastModified); oldDate != "" && newDate != "" {
		parts = append(parts, fmt.Sprintf("(%s → %s)", oldDate, newDate))
	}
	return strings.Join(parts, "  ")
}

// shortRev returns the first 7 chars of the locked git rev, or "" for refs
// without one (path:, tarball:, unlocked refs).
func shortRev(resolved string) string {
	installable, err := flake.ParseInstallable(resolved)
	if err != nil || installable.Ref.Rev == "" {
		return ""
	}
	if len(installable.Ref.Rev) < 7 {
		return installable.Ref.Rev
	}
	return installable.Ref.Rev[:7]
}

func shortDate(rfc3339 string) string {
	if rfc3339 == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
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
