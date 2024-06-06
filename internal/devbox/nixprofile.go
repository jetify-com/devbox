package devbox

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/nix/nixprofile"
)

// syncNixProfileFromFlake ensures the nix profile has the packages from the buildInputs
// from the devshell of the generated flake.
//
// It also removes any packages from the nix profile that are no longer in the buildInputs.
func (d *Devbox) syncNixProfileFromFlake(ctx context.Context) error {
	defer debug.FunctionTimer().End()
	// Get the computed Devbox environment from the generated flake
	env, err := d.computeEnv(ctx, false /*usePrintDevEnvCache*/)
	if err != nil {
		return err
	}

	// Get the store-paths of the packages we want installed in the nix profile
	wantStorePaths := []string{}
	if env["buildInputs"] != "" {
		// env["buildInputs"] can be empty string if there are no packages in the project
		// if buildInputs is empty, then we don't want wantStorePaths to be an array with a single "" entry
		wantStorePaths = strings.Split(env["buildInputs"], " ")
	}

	profilePath, err := d.profilePath()
	if err != nil {
		return err
	}

	// Get the store-paths of the packages currently installed in the nix profile
	items, err := nixprofile.ProfileListItems(d.stderr, profilePath)
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
		debug.Log("Removing packages from nix profile: %s\n", strings.Join(packagesToRemove, ", "))

		if err := nix.ProfileRemove(profilePath, remove...); err != nil {
			return err
		}
	}
	if len(add) > 0 {
		if err = nix.ProfileInstall(ctx, &nix.ProfileInstallArgs{
			Installables: add,
			ProfilePath:  profilePath,
			Writer:       d.stderr,
		}); errors.Is(err, nix.ErrPriorityConflict) {
			// We need to install the packages one by one because there was possibly a priority conflict
			// This is slower, but uncommon.
			for _, addPath := range add {
				if err = nix.ProfileInstall(ctx, &nix.ProfileInstallArgs{
					Installables: []string{addPath},
					ProfilePath:  profilePath,
					Writer:       d.stderr,
				}); err != nil {
					return fmt.Errorf("error installing package in nix profile %s: %w", addPath, err)
				}
			}
		} else if err != nil {
			return fmt.Errorf("error installing packages in nix profile %s: %w", add, err)
		}
	}
	return nil
}
