// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nixprofile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/redact"
)

// ProfileListItems returns a list of the installed packages.
func ProfileListItems(
	writer io.Writer,
	profileDir string,
) (map[string]*NixProfileListItem, error) {

	output, err := nix.ProfileList(writer, profileDir, true /*useJSON*/)
	if err == nil {
		type ProfileListElement struct {
			Active      bool     `json:"active"`
			AttrPath    string   `json:"attrPath"`
			OriginalURL string   `json:"originalUrl"`
			Priority    int      `json:"priority"`
			StorePaths  []string `json:"storePaths"`
			URL         string   `json:"url"`
		}
		type ProfileListOutput struct {
			Elements []ProfileListElement `json:"elements"`
			Version  int                  `json:"version"`
		}

		var structOutput ProfileListOutput
		if err := json.Unmarshal([]byte(output), &structOutput); err != nil {
			return nil, err
		}

		result := map[string]*NixProfileListItem{}
		for index, element := range structOutput.Elements {
			// We use the unlocked reference as the key, since that is the format
			// used for the `nix profile list` output of older nix versions
			// (pre 2.17), which our code is designed to support.
			unlockedReference := element.OriginalURL + "#" + element.AttrPath
			result[unlockedReference] = &NixProfileListItem{
				index:             index,
				unlockedReference: unlockedReference,
				lockedReference:   element.URL + "#" + element.AttrPath,
				nixStorePath:      element.StorePaths[0],
			}
		}
		return result, nil
	}

	output, err = nix.ProfileList(writer, profileDir, false /*useJSON*/)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	lines := strings.Split(output, "\n")

	// The `line` output is of the form:
	// <index> <UnlockedReference> <LockedReference> <NixStorePath>
	//
	// Using an example:
	// 0 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 /nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3
	// 1 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.vim github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.vim /nix/store/gapbqxx1d49077jk8ay38z11wgr12p23-vim-9.0.0609

	items := map[string]*NixProfileListItem{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		item, err := parseNixProfileListItem(line)
		if err != nil {
			return nil, err
		}

		items[item.unlockedReference] = item
	}
	return items, nil
}

type ProfileListIndexArgs struct {
	// For performance you can reuse the same list in multiple operations if you
	// are confident index has not changed.
	List       map[string]*NixProfileListItem
	Lockfile   *lock.File
	Writer     io.Writer
	Input      *devpkg.Package
	ProfileDir string
}

func ProfileListIndex(args *ProfileListIndexArgs) (int, error) {
	var err error
	list := args.List
	if list == nil {
		list, err = ProfileListItems(args.Writer, args.ProfileDir)
		if err != nil {
			return -1, err
		}
	}

	inCache, err := args.Input.IsInBinaryCache()
	if err != nil {
		return -1, err
	}
	if inCache {
		pathInStore, err := args.Input.Installable()
		if err != nil {
			return -1, err
		}
		for _, item := range list {
			if pathInStore == item.nixStorePath {
				return item.index, nil
			}
		}
	}
	// else: fallback to checking if the Input matches an item's unlockedReference

	// This is an optimization for happy path. A resolved devbox package
	// should match the unlockedReference of an existing profile item.
	ref, err := args.Input.NormalizedDevboxPackageReference()
	if err != nil {
		return -1, err
	}
	if item, found := list[ref]; found {
		return item.index, nil
	}

	for _, item := range list {
		existing := item.ToPackage(args.Lockfile)

		if args.Input.Equals(existing) {
			return item.index, nil
		}
	}
	return -1, errors.Wrap(nix.ErrPackageNotFound, args.Input.String())
}

// NixProfileListItem is a go-struct of a line of printed output from `nix profile list`
// docs: https://nixos.org/manual/nix/stable/command-ref/new-cli/nix3-profile-list.html
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

// parseNixProfileListItem reads each line of output (from `nix profile list`) and converts
// into a golang struct. Refer to NixProfileListItem struct definition for explanation of each field.
func parseNixProfileListItem(line string) (*NixProfileListItem, error) {
	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)

	if !scanner.Scan() {
		return nil, redact.Errorf("error parsing \"nix profile list\" output: line is missing index: %s", line)
	}

	index, err := strconv.Atoi(scanner.Text())
	if err != nil {
		return nil, redact.Errorf("error parsing \"nix profile list\" output: %w: %s", err, line)
	}

	if !scanner.Scan() {
		return nil, redact.Errorf("error parsing \"nix profile list\" output: line is missing unlockedReference: %s", line)
	}
	unlockedReference := scanner.Text()

	if !scanner.Scan() {
		return nil, redact.Errorf("error parsing \"nix profile list\" output: line is missing lockedReference: %s", line)
	}
	lockedReference := scanner.Text()

	if !scanner.Scan() {
		return nil, redact.Errorf("error parsing \"nix profile list\" output: line is missing nixStorePath: %s", line)
	}
	nixStorePath := scanner.Text()

	return &NixProfileListItem{
		index:             index,
		unlockedReference: unlockedReference,
		lockedReference:   lockedReference,
		nixStorePath:      nixStorePath,
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
		return "", redact.Errorf(
			"expected to find # in lockedReference: %s from NixProfileListItem: %s",
			redact.Safe(item.lockedReference),
			item,
		)
	}
	return attrPath, nil
}

// ToPackage constructs a nix.Package using the unlocked reference
func (item *NixProfileListItem) ToPackage(locker lock.Locker) *devpkg.Package {
	return devpkg.PackageFromString(item.unlockedReference, locker)
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

type ProfileInstallArgs struct {
	CustomStepMessage string
	Lockfile          *lock.File
	Package           string
	ProfilePath       string
	Writer            io.Writer
}

// ProfileInstall calls nix profile install with default profile
func ProfileInstall(args *ProfileInstallArgs) error {
	input := devpkg.PackageFromString(args.Package, args.Lockfile)

	inCache, err := input.IsInBinaryCache()
	if err != nil {
		return err
	}

	if !inCache && nix.IsGithubNixpkgsURL(input.URLForFlakeInput()) {
		if err := nix.EnsureNixpkgsPrefetched(args.Writer, input.HashFromNixPkgsURL()); err != nil {
			return err
		}
		if exists, err := input.ValidateInstallsOnSystem(); err != nil {
			return err
		} else if !exists {
			return usererr.New(
				"package %s cannot be installed on your system. It may be installable on other systems.",
				input.String(),
			)
		}
	}
	stepMsg := args.Package
	if args.CustomStepMessage != "" {
		stepMsg = args.CustomStepMessage
		// Only print this first one if we have a custom message. Otherwise it feels
		// repetitive.
		fmt.Fprintf(args.Writer, "%s\n", stepMsg)
	}

	installable, err := input.Installable()
	if err != nil {
		return err
	}

	err = nix.ProfileInstall(args.Writer, args.ProfilePath, installable)
	if err != nil {
		fmt.Fprintf(args.Writer, "%s: ", stepMsg)
		color.New(color.FgRed).Fprintf(args.Writer, "Fail\n")
		return redact.Errorf("error running \"nix profile install\": %w", err)
	}

	fmt.Fprintf(args.Writer, "%s: ", stepMsg)
	color.New(color.FgGreen).Fprintf(args.Writer, "Success\n")
	return nil
}

// ProfileRemoveItems removes the items from the profile, in a single call, using their indexes.
// It is up to the caller to ensure that the underlying profile has not changed since the items
// were queried.
func ProfileRemoveItems(profilePath string, items []*NixProfileListItem) error {
	if items == nil {
		return nil
	}
	indexes := []string{}
	for _, item := range items {
		indexes = append(indexes, strconv.Itoa(item.index))
	}
	return nix.ProfileRemove(profilePath, indexes)
}
