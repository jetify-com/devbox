// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nixprofile

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/redact"
)

// ProfileListItems returns a list of the installed packages.
func ProfileListItems(
	writer io.Writer,
	profileDir string,
) ([]*NixProfileListItem, error) {
	defer debug.FunctionTimer().End()
	output, err := nix.ProfileList(writer, profileDir, true /*useJSON*/)
	if err != nil {
		// fallback to legacy profile list
		// NOTE: maybe we should check the nix version first, instead of falling back on _any_ error.
		return profileListLegacy(writer, profileDir)
	}

	type ProfileListElement struct {
		Active      bool     `json:"active"`
		AttrPath    string   `json:"attrPath"`
		OriginalURL string   `json:"originalUrl"`
		Priority    int      `json:"priority"`
		StorePaths  []string `json:"storePaths"`
		URL         string   `json:"url"`
	}
	type ProfileListOutput struct {
		Elements map[string]ProfileListElement `json:"elements"`
		Version  int                           `json:"version"`
	}

	// Modern nix profiles: nix >= 2.20
	var structOutput ProfileListOutput
	if err := json.Unmarshal([]byte(output), &structOutput); err == nil {
		items := []*NixProfileListItem{}
		for name, element := range structOutput.Elements {
			items = append(items, &NixProfileListItem{
				name:              name,
				unlockedReference: lo.Ternary(element.OriginalURL != "", element.OriginalURL+"#"+element.AttrPath, ""),
				lockedReference:   lo.Ternary(element.URL != "", element.URL+"#"+element.AttrPath, ""),
				nixStorePaths:     element.StorePaths,
			})
		}
		return items, nil
	}
	// Fall back to trying format for nix < version 2.20

	// ProfileListOutputJSONLegacy is for parsing `nix profile list --json` in nix < version 2.20
	// that relied on index instead of name for each package installed.
	type ProfileListOutputJSONLegacy struct {
		Elements []ProfileListElement `json:"elements"`
		Version  int                  `json:"version"`
	}
	var structOutput2 ProfileListOutputJSONLegacy
	if err := json.Unmarshal([]byte(output), &structOutput2); err != nil {
		return nil, err
	}
	items := []*NixProfileListItem{}
	for index, element := range structOutput2.Elements {
		items = append(items, &NixProfileListItem{
			index:             index,
			unlockedReference: lo.Ternary(element.OriginalURL != "", element.OriginalURL+"#"+element.AttrPath, ""),
			lockedReference:   lo.Ternary(element.URL != "", element.URL+"#"+element.AttrPath, ""),
			nixStorePaths:     element.StorePaths,
		})
	}

	return items, nil
}

// profileListLegacy lists the items in a nix profile before nix 2.17.0 introduced --json.
func profileListLegacy(
	writer io.Writer,
	profileDir string,
) ([]*NixProfileListItem, error) {
	output, err := nix.ProfileList(writer, profileDir, false /*useJSON*/)
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

	items := []*NixProfileListItem{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		item, err := parseNixProfileListItemLegacy(line)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	return items, nil
}

type ProfileListNameOrIndexArgs struct {
	// For performance, you can reuse the same list in multiple operations if you
	// are confident index has not changed.
	Items      []*NixProfileListItem
	Lockfile   *lock.File
	Writer     io.Writer
	Package    *devpkg.Package
	ProfileDir string
}

// ProfileListNameOrIndex returns the name or index of args.Package in the nix profile specified by args.ProfileDir,
// or nix.ErrPackageNotFound if it's not found. Callers can pass in args.Items to avoid having to call `nix-profile list` again.
func ProfileListNameOrIndex(args *ProfileListNameOrIndexArgs) (string, error) {
	var err error
	items := args.Items
	if items == nil {
		items, err = ProfileListItems(args.Writer, args.ProfileDir)
		if err != nil {
			return "", err
		}
	}

	inCache, err := args.Package.IsInBinaryCache()
	if err != nil {
		return "", err
	}

	if !inCache && args.Package.IsDevboxPackage {
		// This is an optimization for happy path when packages are added by flake reference. A resolved devbox
		// package *which was added by flake reference* (not by store path) should match the unlockedReference
		// of an existing profile item.
		ref, err := args.Package.NormalizedDevboxPackageReference()
		if err != nil {
			return "", errors.Wrapf(err, "failed to get installable for %s", args.Package.String())
		}

		for _, item := range items {
			if ref == item.unlockedReference {
				return item.NameOrIndex(), nil
			}
		}
		return "", errors.Wrap(nix.ErrPackageNotFound, args.Package.String())
	}

	for _, item := range items {
		if item.Matches(args.Package, args.Lockfile) {
			return item.NameOrIndex(), nil
		}
	}
	return "", errors.Wrap(nix.ErrPackageNotFound, args.Package.String())
}

// parseNixProfileListItemLegacy reads each line of output (from `nix profile list`) and converts
// into a golang struct. Refer to NixProfileListItem struct definition for explanation of each field.
// NOTE: this API is for legacy nix. Newer nix versions use --json output.
func parseNixProfileListItemLegacy(line string) (*NixProfileListItem, error) {
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
		return nil, redact.Errorf("error parsing \"nix profile list\" output: line is missing nixStorePaths: %s", line)
	}
	nixStorePaths := strings.Fields(scanner.Text())

	return &NixProfileListItem{
		index:             index,
		unlockedReference: unlockedReference,
		lockedReference:   lockedReference,
		nixStorePaths:     nixStorePaths,
	}, nil
}
