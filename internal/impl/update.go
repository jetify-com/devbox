// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"context"
	"fmt"

	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/wrapnix"
)

func (d *Devbox) Update(ctx context.Context, pkgs ...string) error {
	inputs, err := d.inputsToUpdate(pkgs...)
	if err != nil {
		return err
	}

	for _, pkg := range inputs {
		if pkg.IsLegacy() {
			fmt.Fprintf(d.writer, "Updating %s -> %s\n", pkg.Raw, pkg.LegacyToVersioned())
			if err := d.Remove(ctx, pkg.Raw); err != nil {
				return err
			}
			if err := d.lockfile.ResolveToCurrentNixpkgCommitHash(
				pkg.LegacyToVersioned(),
			); err != nil {
				return err
			}
			if err := d.Add(ctx, pkg.LegacyToVersioned()); err != nil {
				return err
			}
		}
	}

	for _, pkg := range inputs {
		if !lock.IsVersionedPackage(pkg.Raw) {
			fmt.Fprintf(d.writer, "Skipping %s because it is not a versioned package\n", pkg)
			continue
		}
		existing := d.lockfile.Packages[pkg.Raw]
		newEntry, err := searcher.Client().Resolve(pkg.Raw)
		if err != nil {
			return err
		}
		if existing != nil && existing.Version != newEntry.Version {
			fmt.Fprintf(d.writer, "Updating %s %s -> %s\n", pkg, existing.Version, newEntry.Version)
			if err := d.removePackagesFromProfile(ctx, []string{pkg.Raw}); err != nil {
				// Warn but continue. TODO(landau): ensurePackagesAreInstalled should
				// sync the profile so we don't need to do this manually.
				ux.Fwarning(d.writer, "Failed to remove %s from profile: %s\n", pkg, err)
			}
		} else if existing == nil {
			fmt.Fprintf(d.writer, "Resolved %s to %[1]s %[2]s\n", pkg, newEntry.Resolved)
		} else {
			fmt.Fprintf(d.writer, "Already up-to-date %s %s\n", pkg, existing.Version)
		}
		// Set the new entry after we've removed the old package from the profile
		d.lockfile.Packages[pkg.Raw] = newEntry
	}

	// TODO(landau): Improve output
	if err := d.ensurePackagesAreInstalled(ctx, ensure); err != nil {
		return err
	}

	return wrapnix.CreateWrappers(ctx, d)
}

func (d *Devbox) inputsToUpdate(pkgs ...string) ([]*nix.Input, error) {
	var pkgsToUpdate []string
	for _, pkg := range pkgs {
		found, err := d.findPackageByName(pkg)
		if err != nil {
			return nil, err
		}
		pkgsToUpdate = append(pkgsToUpdate, found)
	}
	if len(pkgsToUpdate) == 0 {
		pkgsToUpdate = d.Packages()
	}

	return nix.InputsFromStrings(pkgsToUpdate, d.lockfile), nil
}
