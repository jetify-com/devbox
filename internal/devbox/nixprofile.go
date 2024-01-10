package devbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/nix/nixprofile"
	"go.jetpack.io/devbox/internal/ux"
)

// syncNixProfile ensures the nix profile has the packages specified in wantStorePaths.
// It also removes any packages from the nix profile that are not in wantStorePaths.
func (d *Devbox) syncNixProfile(ctx context.Context, wantStorePaths []string) error {
	profilePath, err := d.profilePath()
	if err != nil {
		return err
	}

	// Get the store-paths of the packages currently installed in the nix profile
	items, err := nixprofile.ProfileListItems(ctx, d.stderr, profilePath)
	if err != nil {
		return fmt.Errorf("nix profile list: %v", err)
	}
	gotStorePaths := make([]string, 0, len(items))
	for _, item := range items {
		gotStorePaths = append(gotStorePaths, item.StorePaths()...)
	}

	// Diff the store paths and install/remove packages as needed
	remove, add := lo.Difference(gotStorePaths, wantStorePaths)
	if len(remove) > 0 {
		packagesToRemove := make([]string, 0, len(remove))
		for _, p := range remove {
			storePath := nix.NewStorePathParts(p)
			packagesToRemove = append(packagesToRemove, fmt.Sprintf("%s@%s", storePath.Name, storePath.Version))
		}
		if len(packagesToRemove) == 1 {
			ux.Finfo(d.stderr, "Removing %s\n", strings.Join(packagesToRemove, ", "))
		} else {
			ux.Finfo(d.stderr, "Removing packages: %s\n", strings.Join(packagesToRemove, ", "))
		}

		if err := nix.ProfileRemove(profilePath, remove...); err != nil {
			return err
		}
	}
	if len(add) > 0 {
		total := len(add)
		for idx, addPath := range add {
			stepNum := idx + 1
			storePath := nix.NewStorePathParts(addPath)
			nameAndVersion := fmt.Sprintf("%s@%s", storePath.Name, storePath.Version)
			stepMsg := fmt.Sprintf("[%d/%d] %s", stepNum, total, nameAndVersion)

			if err = nixprofile.ProfileInstall(ctx, &nixprofile.ProfileInstallArgs{
				CustomStepMessage: stepMsg,
				Installable:       addPath,
				// Install in offline mode for speed. We know we should have all the files
				// locally in /nix/store since we have run `nix print-dev-env` prior to this.
				// Also avoids some "substituter not found for store-path" errors.
				Offline:     true,
				PackageName: storePath.Name,
				ProfilePath: profilePath,
				Writer:      d.stderr,
			}); err != nil {
				return fmt.Errorf("error installing package %s: %w", addPath, err)
			}
		}
	}
	return nil
}
