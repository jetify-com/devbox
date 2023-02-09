package nix

import (
	"fmt"
	"testing"
)

func TestNixProfileListItem(t *testing.T) {

	line := fmt.Sprintf("%d %s %s %s",
		0,
		"github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19",
		"github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19",
		"/nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3",
	)
	item, err := parseNixProfileListItem(line)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if item == nil {
		t.Fatalf("expected NixProfileListItem to be non-nil")
	}

	expected := &NixProfileListItem{
		index:             0,
		unlockedReference: "github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19",
		lockedReference:   "github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19",
		nixStorePath:      "/nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3",
	}
	if *item != *expected {
		t.Fatalf("expected parsed NixProfileListItem to be %s but got %s",
			expected,
			item,
		)
	}

	gotAttrPath, err := item.AttributePath()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	wantAttrPath := "legacyPackages.x86_64-darwin.go_1_19"
	if gotAttrPath != wantAttrPath {
		t.Errorf("expected attribute path %s but got %s", wantAttrPath, gotAttrPath)
	}

	gotPkgName, err := item.PackageName()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	wantPkgName := "go_1_19"
	if gotPkgName != wantPkgName {
		t.Errorf("expected package name %s but got %s", wantPkgName, gotPkgName)
	}
}
