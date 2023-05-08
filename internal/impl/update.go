package impl

import (
	"context"
	"fmt"

	"go.jetpack.io/devbox/internal/lock"
)

func (d *Devbox) Update(ctx context.Context, pkgs ...string) error {
	var pkgsToUpdate []string
	for _, pkg := range pkgs {
		found, err := d.findPackageByName(pkg)
		if err != nil {
			return err
		}
		pkgsToUpdate = append(pkgsToUpdate, found)
	}
	if len(pkgsToUpdate) == 0 {
		pkgsToUpdate = d.packages()
	}

	for _, pkg := range pkgsToUpdate {
		if !lock.IsVersionedPackage(pkg) {
			fmt.Fprintf(d.writer, "Skipping %s because it is not a versioned package\n", pkg)
			continue
		}
		existing := d.lockfile.Packages[pkg]
		newEntry, err := d.lockfile.ForceResolve(pkg)
		if err != nil {
			return err
		}
		if existing != nil && existing.Version != newEntry.Version {
			fmt.Fprintf(d.writer, "Updating %s %s -> %s\n", pkg, existing.Version, newEntry.Version)
			if err := d.removePackagesFromProfile(ctx, []string{pkg}); err != nil {
				return err
			}
		} else if existing == nil {
			fmt.Fprintf(d.writer, "Resolved %s to %[1]s %[2]s\n", pkg, newEntry.Resolved)
		} else {
			fmt.Fprintf(d.writer, "Already up-to-date %s %s\n", pkg, existing.Version)
		}
	}

	// TODO(landau): Improve output
	return d.ensurePackagesAreInstalled(ctx, ensure)
}
