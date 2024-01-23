package nixprofile

import (
	"fmt"
	"strings"

	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/redact"
)

// NixProfileListItem is a go-struct of a line of printed output from `nix profile list`
// docs: https://nixos.org/manual/nix/stable/command-ref/new-cli/nix3-profile-list.html
type NixProfileListItem struct {
	// An integer that can be used to unambiguously identify the package in
	// invocations of nix profile remove and nix profile upgrade.
	index int

	// The original ("unlocked") flake reference and output attribute path used at installation time.
	// NOTE that this will be empty if the package was added to the nix profile via store path.
	unlockedReference string

	// The locked flake reference to which the unlocked flake reference was resolved.
	// NOTE that this will be empty if the package was added to the nix profile via store path.
	lockedReference string

	// The store path(s) of the package. Should have at least 1 path, and should have exactly 1 path
	// if the item was added to the profile through a store path.
	nixStorePaths []string
}

// AttributePath parses the package attribute from the NixProfileListItem.lockedReference
//
// For example:
// if NixProfileListItem.lockedReference = github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
// then AttributePath = legacyPackages.x86_64-darwin.go_1_19
func (i *NixProfileListItem) AttributePath() (string, error) {
	// lockedReference example:
	// github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19

	// AttributePath example:
	// legacyPackages.x86_64.go_1_19
	_ /*nixpkgs*/, attrPath, found := strings.Cut(i.lockedReference, "#")
	if !found {
		return "", redact.Errorf(
			"expected to find # in lockedReference: %s from NixProfileListItem: %s",
			redact.Safe(i.lockedReference),
			i,
		)
	}
	return attrPath, nil
}

// Matches compares a devpkg.Package with this profile item and returns true if the profile item
// was the result of adding the Package to the nix profile.
func (i *NixProfileListItem) Matches(pkg *devpkg.Package, locker lock.Locker) bool {
	if i.addedByStorePath() {
		// If an Item was added via store path, the best we can do when comparing to a Package is to check
		// if its store path matches that of the Package. Note that the item should only have 1 store path.
		path, err := pkg.InputAddressedPath()
		if err != nil {
			// pkg couldn't have been added by store path if we can't get the store path for it, so return
			// false. There are some edge cases (e.g. cache is down, index changed, etc., but it's OK to
			// err on the side of false).
			return false
		}
		return len(i.nixStorePaths) == 1 && i.nixStorePaths[0] == path
	}

	return pkg.Equals(devpkg.PackageFromStringWithDefaults(i.unlockedReference, locker))
}

func (i *NixProfileListItem) addedByStorePath() bool {
	return i.unlockedReference == ""
}

// String serializes the NixProfileListItem back into the format printed by `nix profile list`
func (i *NixProfileListItem) String() string {
	return fmt.Sprintf("{%d %s %s %s}",
		i.index,
		i.unlockedReference,
		i.lockedReference,
		i.nixStorePaths,
	)
}
