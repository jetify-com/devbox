package devbox

import (
	"context"
	"fmt"
	"strings"

	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/nix/nixprofile"
)

// syncFlakeToProfile ensures the buildInputs from the flake's devShell are
// installed in the nix profile.
// buildInputs is a space-separated list of store paths from the nix print-dev-env output's buildInputs.
func (d *Devbox) syncFlakeToProfile(ctx context.Context, buildInputs string) error {
	profilePath, err := d.profilePath()
	if err != nil {
		return err
	}

	// Get the build inputs (i.e. store paths) from the generated flake's print-dev-env output
	wantStorePaths := []string{}
	if buildInputs != "" { // if buildInputs is empty, then we don't want wantStorePaths to be an array with a single "" entry
		wantStorePaths = strings.Split(buildInputs, " ")
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
	add, remove := diffStorePaths(gotStorePaths, wantStorePaths)
	if len(remove) > 0 {
		packagesToRemove := make([]string, 0, len(remove))
		for _, p := range remove {
			storePath := nix.NewStorePathParts(p)
			packagesToRemove = append(packagesToRemove, fmt.Sprintf("%s@%s", storePath.Name, storePath.Version))
		}
		if len(packagesToRemove) == 1 {
			fmt.Fprintf(d.stderr, "Removing %s\n", strings.Join(packagesToRemove, ", "))
		} else {
			fmt.Fprintf(d.stderr, "Removing packages: %s\n", strings.Join(packagesToRemove, ", "))
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

func diffStorePaths(got, want []string) (add, remove []string) {
	gotSet := map[string]bool{}
	for _, g := range got {
		gotSet[g] = true
	}
	wantSet := map[string]bool{}
	for _, w := range want {
		wantSet[w] = true
	}

	for _, g := range got {
		if _, ok := wantSet[g]; !ok {
			remove = append(remove, g)
		}
	}

	for _, w := range want {
		if _, ok := gotSet[w]; !ok {
			add = append(add, w)
		}
	}
	return add, remove
}
