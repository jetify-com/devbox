package nix

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
)

// ProfileListItems returns a list of the installed packages
func ProfileListItems(writer io.Writer, profileDir string) ([]*NixProfileListItem, error) {
	if featureflag.Flakes.Disabled() {
		return nil, errors.New("Not supported for legacy non-flakes implementation")
	}

	cmd := exec.Command(
		"nix", "profile", "list",
		"--extra-experimental-features", "nix-command flakes",
		"--profile", profileDir,
	)

	// We set stderr to a different output than stdout
	// to ensure error output is not mingled with the stdout output
	// that we need to parse.
	cmd.Stderr = writer

	out, err := cmd.Output()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// The `out` output is of the form:
	// <index> <UnlockedReference> <LockedReference> <NixStorePath>
	//
	// Using an example:
	// 0 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 /nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3
	// 1 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.vim github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.vim /nix/store/gapbqxx1d49077jk8ay38z11wgr12p23-vim-9.0.0609
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	items := []*NixProfileListItem{}
	for _, line := range lines {
		item, err := ParseNixProfileListItemIfAny(line)
		if err != nil {
			return nil, err
		}
		if item == nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// NixProfileListItem is a go-struct of a line of printed output from `nix profile list`
type NixProfileListItem struct {
	// An integer that can be used to unambiguously identify the package in
	// invocations of nix profile remove and nix profile upgrade.
	index int

	// The original ("unlocked") flake reference and output attribute path used at installation time.
	unlockedReference string

	// The locked flake reference to which the unlocked flake reference was resolved.
	lockedReference string

	// The store path(s) of the package.
	nixStorePath string
}

func ParseNixProfileListItemIfAny(line string) (*NixProfileListItem, error) {
	// line is a line of printed output from `nix profile list`
	//
	// line example:
	// 0 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 /nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	// parts example:
	// [
	//   0
	//   github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
	//   github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
	//   /nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3
	// ]
	parts := strings.Split(line, " ")
	if len(parts) != 4 {
		return nil, errors.Errorf(
			"Expected 4 parts for line in nix profile list, but got %d parts. Line: %s",
			len(parts),
			line,
		)
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &NixProfileListItem{
		index:             index,
		unlockedReference: parts[1],
		lockedReference:   parts[2],
		nixStorePath:      parts[3],
	}, nil
}

// AttributePath parses the package attribute from the NixProfileListItem.lockedReference
//
// For example:
// if NixProfileListItem.lockedReference = github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
// then AttributePath = legacyPackages.x86_64-darwin.go_1_19
func (item *NixProfileListItem) AttributePath() (string, error) {

	// lockedReference example:
	// github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19

	// AttributePath example:
	// legacyPackages.x86_64.go_1_19
	_ /*nixpkgs*/, attrPath, found := strings.Cut(item.lockedReference, "#")
	if !found {
		return "", errors.Errorf(
			"expected to find # in lockedReference: %s from NixProfileListItem: %s",
			item.lockedReference,
			item.String(),
		)
	}
	return attrPath, nil
}

// PackageName parses the package name from the NixProfileListItem.lockedReference
//
// For example:
// if NixProfileListItem.lockedReference = github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
// then AttributePath = legacyPackages.x86_64-darwin.go_1_19
// and then PackageName = go_1_19
func (item *NixProfileListItem) PackageName() (string, error) {
	attrPath, err := item.AttributePath()
	if err != nil {
		return "", err
	}

	parts := strings.Split(attrPath, ".")
	if len(parts) < 2 {
		return "", errors.Errorf(
			"Expected >= 2 parts for AttributePath in nix profile list, but got %d parts. AttributePath: %s",
			len(parts),
			attrPath,
		)
	}

	packageName := parts[len(parts)-1]
	return packageName, nil
}

// String serializes the NixProfileListItem back into the format printed by `nix profile list`
func (item *NixProfileListItem) String() string {
	return fmt.Sprintf("%d %s %s %s",
		item.index,
		item.unlockedReference,
		item.lockedReference,
		item.nixStorePath,
	)
}
