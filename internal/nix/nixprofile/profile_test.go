// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nixprofile

import (
	"fmt"
	"reflect"
	"testing"
)

type expectedTestData struct {
	item        *NixProfileListItem
	attrPath    string
	packageName string
}

// TestNixProfileListItemLegacy tests the parsing of legacy nix profile list items.
// It only applies to much older nix versions. Newer nix versions rely on the --json output
// instead parsing the legacy output.
func TestNixProfileListItemLegacy(t *testing.T) {
	testCases := map[string]struct {
		line     string
		expected expectedTestData
	}{
		"go_1_19": {
			line: fmt.Sprintf(
				"%d %s %s %s",
				0,
				"flake:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19",
				"github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19",
				"/nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3",
			),
			expected: expectedTestData{
				item: &NixProfileListItem{
					index:             0,
					unlockedReference: "flake:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19",
					lockedReference:   "github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19",
					nixStorePaths:     []string{"/nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3"},
				},
				attrPath:    "legacyPackages.x86_64-darwin.go_1_19",
				packageName: "go_1_19",
			},
		},
		"numpy": {
			line: fmt.Sprintf("%d %s %s %s",
				2,
				"github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.python39Packages.numpy",
				"github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin."+
					"python39Packages.numpy ",
				"/nix/store/qly36iy1p4q1h5p4rcbvsn3ll0zsd9pd-python3.9-numpy-1.23.3",
			),
			expected: expectedTestData{
				item: &NixProfileListItem{
					index:             2,
					unlockedReference: "github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.python39Packages.numpy",
					lockedReference:   "github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.python39Packages.numpy",
					nixStorePaths:     []string{"/nix/store/qly36iy1p4q1h5p4rcbvsn3ll0zsd9pd-python3.9-numpy-1.23.3"},
				},
				attrPath:    "legacyPackages.x86_64-darwin.python39Packages.numpy",
				packageName: "python39Packages.numpy",
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			testItem(t, testCase.line, testCase.expected)
		})
	}
}

func testItem(t *testing.T, line string, expected expectedTestData) {
	item, err := parseNixProfileListItemLegacy(line)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if item == nil {
		t.Fatalf("expected NixProfileListItem to be non-nil")
	}

	if !reflect.DeepEqual(item, expected.item) {
		t.Fatalf("expected parsed NixProfileListItem to be %s but got %s",
			expected.item,
			item,
		)
	}

	gotAttrPath, err := item.AttributePath()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if gotAttrPath != expected.attrPath {
		t.Errorf("expected attribute path %s but got %s", expected.attrPath, gotAttrPath)
	}
}
