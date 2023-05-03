package impl

import (
	"context"
	"fmt"
)

func (d *Devbox) Update(ctx context.Context, pkgs ...string) error {
	pkgsToUpdate := []string{}
	if len(pkgs) == 0 {
		pkgsToUpdate = append([]string(nil), d.packages()...)
	} else {
		for _, pkg := range pkgs {
			found, err := d.findPackageByName(pkg)
			if err != nil {
				return err
			}
			pkgsToUpdate = append(pkgsToUpdate, found)
		}
	}

	for _, pkg := range pkgsToUpdate {
		if !d.lockfile.IsVersionedPackage(pkg) {
			fmt.Fprintf(d.writer, "Skipping %s because it is not a versioned package\n", pkg)
			continue
		}
		existing := d.lockfile.Entry(pkg)
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
