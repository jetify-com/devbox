// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nixprofile

import (
	"bufio"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetify.com/devbox/internal/debug"
	"go.jetify.com/devbox/internal/devpkg"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/internal/redact"
)

// ProfileListItems returns a list of the installed packages.
//
// It reads the profile's manifest.json directly when it can, which avoids
// starting a nix process. Otherwise it runs `nix profile list`.
func ProfileListItems(
	writer io.Writer,
	profileDir string,
) ([]*NixProfileListItem, error) {
	defer debug.FunctionTimer().End()
	if items, ok := readManifestItems(profileDir); ok {
		return items, nil
	}
	return profileListCommand(writer, profileDir)
}

// profileListCommand is a variable so tests can check when ProfileListItems
// falls back to running nix.
var profileListCommand = profileListItemsFromCommand

// profileListElement is an element of a profile's manifest.json, which is
// also what `nix profile list --json` prints.
type profileListElement struct {
	// Active is a pointer so parseManifest can tell when it's missing.
	Active      *bool    `json:"active"`
	AttrPath    string   `json:"attrPath"`
	OriginalURL string   `json:"originalUrl"`
	Priority    int      `json:"priority"`
	StorePaths  []string `json:"storePaths"`
	URL         string   `json:"url"`
}

func (e *profileListElement) item(name string, index int) *NixProfileListItem {
	return &NixProfileListItem{
		index:             index,
		name:              name,
		unlockedReference: lo.Ternary(e.OriginalURL != "", e.OriginalURL+"#"+e.AttrPath, ""),
		lockedReference:   lo.Ternary(e.URL != "", e.URL+"#"+e.AttrPath, ""),
		nixStorePaths:     e.StorePaths,
	}
}

// readManifestItems reads the items in a profile from its manifest.json. It
// returns false if the caller should run `nix profile list` instead.
func readManifestItems(profileDir string) ([]*NixProfileListItem, bool) {
	data, err := os.ReadFile(filepath.Join(profileDir, "manifest.json"))
	if errors.Is(err, fs.ErrNotExist) {
		// Like nix, treat a profile without a manifest as empty. This
		// includes a profile that doesn't exist yet and a link to a
		// generation that was garbage collected. A manifest.nix means the
		// profile was made by nix-env, so let nix read that.
		_, err := os.Stat(filepath.Join(profileDir, "manifest.nix"))
		if errors.Is(err, fs.ErrNotExist) {
			return []*NixProfileListItem{}, true
		}
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	return parseManifest(data)
}

// parseManifest parses a version 3 manifest.json (nix >= 2.20) into the same
// items that parsing `nix profile list --json` would give. That command reads
// the manifest and prints it back out in the same format.
//
// It returns false for other versions and for anything nix would handle
// differently. Versions 1 and 2 store elements in an array, and nix >= 2.20
// derives a name for each one when it reads them. Rather than copy that
// logic, those are left to nix. They're rewritten as version 3 the next time
// nix changes the profile.
func parseManifest(data []byte) ([]*NixProfileListItem, bool) {
	var manifest struct {
		Version  int                           `json:"version"`
		Elements map[string]profileListElement `json:"elements"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil || manifest.Version != 3 {
		return nil, false
	}
	items := make([]*NixProfileListItem, 0, len(manifest.Elements))
	for name, element := range manifest.Elements {
		if element.Active == nil {
			// nix fails to read an element without "active".
			return nil, false
		}
		if element.URL == "" {
			// nix ignores the flake source when there's no url, so the
			// command doesn't print originalUrl or attrPath either.
			element.OriginalURL = ""
			element.AttrPath = ""
		}
		// nix keeps store paths in a sorted set and always prints the list.
		if element.StorePaths == nil {
			element.StorePaths = []string{}
		}
		slices.Sort(element.StorePaths)
		element.StorePaths = slices.Compact(element.StorePaths)
		items = append(items, element.item(name, 0))
	}
	return items, true
}

// profileListItemsFromCommand runs `nix profile list` to get the items in a
// profile.
func profileListItemsFromCommand(
	writer io.Writer,
	profileDir string,
) ([]*NixProfileListItem, error) {
	output, err := nix.ProfileList(writer, profileDir, true /*useJSON*/)
	if err != nil {
		// fallback to legacy profile list
		// NOTE: maybe we should check the nix version first, instead of falling back on _any_ error.
		return profileListLegacy(writer, profileDir)
	}

	type ProfileListOutput struct {
		Elements map[string]profileListElement `json:"elements"`
		Version  int                           `json:"version"`
	}

	// Modern nix profiles: nix >= 2.20
	var structOutput ProfileListOutput
	if err := json.Unmarshal([]byte(output), &structOutput); err == nil {
		items := []*NixProfileListItem{}
		for name, element := range structOutput.Elements {
			items = append(items, element.item(name, 0))
		}
		return items, nil
	}
	// Fall back to trying format for nix < version 2.20

	// ProfileListOutputJSONLegacy is for parsing `nix profile list --json` in nix < version 2.20
	// that relied on index instead of name for each package installed.
	type ProfileListOutputJSONLegacy struct {
		Elements []profileListElement `json:"elements"`
		Version  int                  `json:"version"`
	}
	var structOutput2 ProfileListOutputJSONLegacy
	if err := json.Unmarshal([]byte(output), &structOutput2); err != nil {
		return nil, err
	}
	items := []*NixProfileListItem{}
	for index, element := range structOutput2.Elements {
		items = append(items, element.item("", index))
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
