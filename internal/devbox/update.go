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
	"go.jetify.com/devbox/internal/ux"
	"go.jetify.com/devbox/nix/flake"
	"golang.org/x/sync/errgroup"
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

	// Legacy (unversioned) packages are migrated to versioned ones by
	// replacing them in the config. This only edits the config; packages are
	// installed once by the ensureStateIsUpToDate call below.
	pendingPackagesToUpdate := []*devpkg.Package{}
	var migrated []*addedPackages
	for _, pkg := range inputs {
		if !pkg.IsLegacy() {
			pendingPackagesToUpdate = append(pendingPackagesToUpdate, pkg)
			continue
		}
		fmt.Fprintf(d.stderr, "Updating %s -> %s\n", pkg.Raw, pkg.LegacyToVersioned())

		// Get the package from the config to get the Platforms and ExcludedPlatforms later
		cfgPackage, ok := d.cfg.Root.GetPackage(pkg.Raw)
		if !ok {
			return fmt.Errorf("package %s not found in config", pkg.Raw)
		}

		if err := d.removeFromConfig(pkg.Raw); err != nil {
			return err
		}
		// Calling addToConfig with the original package names, since it will
		// automatically append @latest if search is able to handle that.
		// If not, it will fallback to the nixpkg format.
		added, err := d.addToConfig(ctx, []string{pkg.Raw}, devopt.AddOpts{
			Platforms:        cfgPackage.Platforms,
			ExcludePlatforms: cfgPackage.ExcludedPlatforms,
		})
		if err != nil {
			return err
		}
		if slices.Contains(added.addedNames, pkg.Raw) {
			// Re-added under the same (unversioned) name: resolve it fresh.
			d.forgetLockEntry(pkg.Raw)
		}
		migrated = append(migrated, added)
	}

	changed, err := d.updatePendingPackages(pendingPackagesToUpdate)
	if err != nil {
		return err
	}

	// Only packages whose lockfile entry changed are rebuilt with --refresh.
	// Unchanged packages are built only if they're missing from the store.
	d.packagesBeingUpdated = changed

	mode := update
	if opts.NoInstall {
		mode = noInstall
	}
	if err := d.ensureStateIsUpToDate(ctx, mode); err != nil {
		return err
	}

	if len(migrated) > 0 {
		if err := d.saveCfg(); err != nil {
			return err
		}
		for _, added := range migrated {
			if err := d.printPostAddMessage(ctx, added.pkgs, added.unchangedPackageNames, added.opts); err != nil {
				return err
			}
		}
	}

	// With --no-install, skip the post-update "nix path-info" (via
	// FixMissingStorePaths). This keeps `devbox update --no-install` usable in
	// environments where Nix isn't fully set up (e.g. CI/Renovate), where that
	// call would otherwise fail. Earlier steps in the update flow may still
	// resolve flake refs via Nix; --no-install only avoids install-like steps.
	// See jetify-com/devbox#2585.
	if !opts.NoInstall {
		// fix any missing store paths.
		if err = d.FixMissingStorePaths(ctx); err != nil {
			return errors.WithStack(err)
		}
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

// maxConcurrentResolves bounds how many packages `devbox update` resolves at
// once. Each resolve is a search API request or a `nix flake metadata` call.
const maxConcurrentResolves = 8

// updatePendingPackages updates the lockfile entries for each package, using
// the right strategy per package kind. Flake refs warn-and-continue on
// failure (see #1180 / #1840); versioned nixpkgs packages abort the update on
// failure. Unversioned non-flake entries are left alone.
//
// Packages are resolved concurrently, then merged into the lockfile serially
// in input order so output and error behavior match a serial update. It
// returns the packages whose lockfile entry changed.
func (d *Devbox) updatePendingPackages(pkgs []*devpkg.Package) ([]*devpkg.Package, error) {
	// refresh=true so flake refs bypass nix's own metadata cache and re-query
	// upstream. Without this, `devbox update` on a github: ref can return a
	// stale commit that nix had cached from an earlier call.
	return d.resolveAndMergePackages(pkgs, func(pkg *devpkg.Package) (*lock.Package, error) {
		return d.lockfile.FetchResolvedPackage(pkg.Raw, true)
	})
}

// resolveAndMergePackages is updatePendingPackages with the resolver injected
// for testing.
func (d *Devbox) resolveAndMergePackages(
	pkgs []*devpkg.Package,
	fetch func(*devpkg.Package) (*lock.Package, error),
) ([]*devpkg.Package, error) {
	type result struct {
		resolved *lock.Package
		err      error
	}
	results := make([]*result, len(pkgs))

	var group errgroup.Group
	group.SetLimit(maxConcurrentResolves)
	for i, pkg := range pkgs {
		if !shouldUpdateLockEntry(pkg) {
			continue
		}
		group.Go(func() error {
			resolved, err := fetch(pkg)
			results[i] = &result{resolved: resolved, err: err}
			return nil
		})
	}
	_ = group.Wait() // errors are carried in results

	var changed []*devpkg.Package
	for i, pkg := range pkgs {
		res := results[i]
		if res == nil {
			continue
		}
		if res.err != nil {
			if pkgtype.IsFlake(pkg.Raw) {
				ux.Fwarningf(d.stderr, "Failed to update %s: %s\n", pkg.Raw, res.err)
				continue
			}
			return nil, res.err
		}
		if res.resolved == nil {
			continue
		}
		before := d.lockfile.Packages[pkg.Raw]
		if err := d.mergeResolvedPackageToLockfile(pkg, res.resolved, d.lockfile); err != nil {
			return nil, err
		}
		// Merging replaces the entry only when something changed.
		if d.lockfile.Packages[pkg.Raw] != before {
			changed = append(changed, pkg)
		}
	}
	return changed, nil
}

// shouldUpdateLockEntry reports whether `devbox update` re-resolves pkg's
// lockfile entry: flake refs and versioned packages.
func shouldUpdateLockEntry(pkg *devpkg.Package) bool {
	if pkgtype.IsFlake(pkg.Raw) {
		return true
	}
	_, _, isVersioned := searcher.ParseVersionedPackage(pkg.Raw)
	return isVersioned
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

	// Skip the guard if either side is missing a timestamp — treat unknown as
	// not-older so we don't block a legit update.
	if existing.LastModified != "" && resolved.LastModified != "" &&
		existing.LastModified > resolved.LastModified {
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
