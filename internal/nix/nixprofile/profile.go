// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nixprofile

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/redact"
)

// ProfileListItems returns a list of the installed packages.
// TODO drop the Profile prefix from this function name.
func ProfileListItems(
	ctx context.Context,
	writer io.Writer,
	profileDir string,
) ([]*NixProfileListItem, error) {
	output, err := nix.ProfileList(ctx, writer, profileDir, true /*useJSON*/)
	if err != nil {
		// fallback to legacy profile list
		// NOTE: maybe we should check the nix version first, instead of falling back on _any_ error.
		return profileListLegacy(ctx, writer, profileDir)
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
		Elements []ProfileListElement `json:"elements"`
		Version  int                  `json:"version"`
	}

	var structOutput ProfileListOutput
	if err := json.Unmarshal([]byte(output), &structOutput); err != nil {
		return nil, err
	}

	items := []*NixProfileListItem{}
	for index, element := range structOutput.Elements {
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
// TODO drop the Profile prefix from this function name.
func profileListLegacy(
	ctx context.Context,
	writer io.Writer,
	profileDir string,
) ([]*NixProfileListItem, error) {
	output, err := nix.ProfileList(ctx, writer, profileDir, false /*useJSON*/)
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
		item, err := parseNixProfileListItem(line)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	return items, nil
}

// TODO drop the Profile prefix from this name.
type ProfileListIndexArgs struct {
	// For performance, you can reuse the same list in multiple operations if you
	// are confident index has not changed.
	Items      []*NixProfileListItem
	Lockfile   *lock.File
	Writer     io.Writer
	Package    *devpkg.Package
	ProfileDir string
}

// ProfileListIndex returns the index of args.Package in the nix profile specified by args.ProfileDir,
// or -1 if it's not found. Callers can pass in args.Items to avoid having to call `nix-profile list` again.
// TODO drop the Profile prefix from this name.
func ProfileListIndex(ctx context.Context, args *ProfileListIndexArgs) (int, error) {
	var err error
	items := args.Items
	if items == nil {
		items, err = ProfileListItems(ctx, args.Writer, args.ProfileDir)
		if err != nil {
			return -1, err
		}
	}

	inCache, err := args.Package.IsInBinaryCache()
	if err != nil {
		return -1, err
	}

	if !inCache && args.Package.IsDevboxPackage {
		// This is an optimization for happy path when packages are added by flake reference. A resolved devbox
		// package *which was added by flake reference* (not by store path) should match the unlockedReference
		// of an existing profile item.
		ref, err := args.Package.NormalizedDevboxPackageReference()
		if err != nil {
			return -1, errors.Wrapf(err, "failed to get installable for %s", args.Package.String())
		}

		for _, item := range items {
			if ref == item.unlockedReference {
				return item.index, nil
			}
		}
		return -1, errors.Wrap(nix.ErrPackageNotFound, args.Package.String())
	}

	for _, item := range items {
		if item.Matches(args.Package, args.Lockfile) {
			return item.index, nil
		}
	}
	return -1, errors.Wrap(nix.ErrPackageNotFound, args.Package.String())
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

// TODO drop the Profile prefix from this name.
type ProfileInstallPackageArgs struct {
	CustomStepMessage string
	Lockfile          *lock.File
	Package           string
	ProfilePath       string
	Writer            io.Writer
}

// ProfileInstallPackage installs a Devbox package into a profile.
// TODO drop the Profile prefix from this name.
func ProfileInstallPackage(ctx context.Context, args *ProfileInstallPackageArgs) error {
	input := devpkg.PackageFromStringWithDefaults(args.Package, args.Lockfile)

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
			platform := nix.System()
			return usererr.New(
				"package %s cannot be installed on your platform %s.\n"+
					"If you know this package is incompatible with %[2]s, then "+
					"you could run `devbox add %[1]s --exclude-platform %[2]s` and re-try.\n"+
					"If you think this package should be compatible with %[2]s, then "+
					"it's possible this particular version is not available yet from the nix registry. "+
					"You could try `devbox add` with a different version for this package.\n",
				input.Raw,
				platform,
			)
		}
	}

	installable, err := input.Installable()
	if err != nil {
		return err
	}
	return ProfileInstall(ctx, &ProfileInstallArgs{
		CustomStepMessage: args.CustomStepMessage,
		Installable:       installable,
		PackageName:       args.Package,
		ProfilePath:       args.ProfilePath,
	})
}

// TODO drop the Profile prefix from this name.
type ProfileInstallArgs struct {
	CustomStepMessage string
	Installable       string
	Offline           bool
	PackageName       string
	ProfilePath       string
	Writer            io.Writer
}

// TODO drop the Profile prefix from this name.
func ProfileInstall(ctx context.Context, args *ProfileInstallArgs) error {
	stepMsg := args.PackageName
	if args.CustomStepMessage != "" {
		stepMsg = args.CustomStepMessage
		// Only print this first one if we have a custom message. Otherwise it feels
		// repetitive.
		fmt.Fprintf(args.Writer, "%s\n", stepMsg)
	}

	err := nix.ProfileInstall(ctx, &nix.ProfileInstallArgs{
		Installable: args.Installable,
		Offline:     args.Offline,
		ProfilePath: args.ProfilePath,
		Writer:      args.Writer,
	})
	if err != nil {
		fmt.Fprintf(args.Writer, "%s: ", stepMsg)
		color.New(color.FgRed).Fprintf(args.Writer, "Fail\n")
		return redact.Errorf("error running \"nix profile install\": %w", err)
	}

	fmt.Fprintf(args.Writer, "%s: ", stepMsg)
	color.New(color.FgGreen).Fprintf(args.Writer, "Success\n")
	return nil
}
