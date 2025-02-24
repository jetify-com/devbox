package nixprofile

import (
	"fmt"
	"strings"

	"go.jetify.com/devbox/internal/devpkg"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/internal/redact"
)

// NixProfileListItem is a go-struct of a line of printed output from `nix profile list`
// docs: https://nixos.org/manual/nix/stable/command-ref/new-cli/nix3-profile-list.html
type NixProfileListItem struct {
	// An integer that can be used to unambiguously identify the package in
	// invocations of nix profile remove and nix profile upgrade.
	index int

	// name of the package
	// nix 2.20 introduced a new format for the output of nix profile list, which includes the package name.
	// This field is used instead of index for `list`, `remove` and `upgrade` subcommands of `nix profile`.
	name string

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
		paths, err := pkg.InputAddressedPaths()
		if err != nil {
			// pkg couldn't have been added by store path if we can't get the store path for it, so return
			// false. There are some edge cases (e.g. cache is down, index changed, etc., but it's OK to
			// err on the side of false).
			return false
		}
		for _, path := range paths {
			return len(i.nixStorePaths) == 1 && i.nixStorePaths[0] == path
		}
		return false
	}

	return pkg.Equals(devpkg.PackageFromStringWithDefaults(i.unlockedReference, locker))
}

func (i *NixProfileListItem) MatchesUnlockedReference(installable string) bool {
	return i.unlockedReference == installable
}

func (i *NixProfileListItem) addedByStorePath() bool {
	return i.unlockedReference == ""
}

// String serializes the NixProfileListItem for debuggability
func (i *NixProfileListItem) String() string {
	return fmt.Sprintf("{nameOrIndex:%s unlockedRef:%s lockedRef:%s, nixStorePaths:%s}",
		i.NameOrIndex(),
		i.unlockedReference,
		i.lockedReference,
		i.nixStorePaths,
	)
}

func (i *NixProfileListItem) StorePaths() []string {
	return i.nixStorePaths
}

// NameOrIndex is a helper method to get the name of the package if it exists, or the index if it doesn't.
// `nix profile` subcommands `list`, `remove`, and `upgrade` use either name (nix >= 2.20) or index (nix < 2.20)
// to identify the package.
func (i *NixProfileListItem) NameOrIndex() string {
	if i.name != "" {
		return i.name
	}
	return fmt.Sprintf("%d", i.index)
}
