// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nixprofile

import (
	"cmp"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"go.jetify.com/devbox/internal/nix"
)

const (
	nixpkgsURL = "https://releases.nixos.org/nixpkgs/nixpkgs-26.11pre1104102.e62746b0b472/nixexprs.tar.zst?narHash=sha256-GYpArmwi/iMuX2ov31oCBhAXRMJdNX9kOl2LZHCuR6M%3D"
	fdPath     = "/nix/store/vx7li94vvv19qqs8dil7vx7fl4by5dqg-fd-10.5.0"
	helloPath  = "/nix/store/fv6rgzamfwj334kf1xv8ms3y6x8nvirn-hello-2.12.3"
	jqBinPath  = "/nix/store/vvp6grz676r5jvsry7sf7kxqb4lwdd62-jq-1.8.2-bin"
	jqManPath  = "/nix/store/makwmvlz85ngimxrznbisy864hvc3g0s-jq-1.8.2-man"
)

// manifestV3 is a manifest written by nix 2.30 for a profile with a package
// added by store path (which is what devbox does), a package added by flake
// reference, and a package with two outputs. The jq store paths are out of
// order to check that they get sorted like nix does.
const manifestV3 = `{"elements":{` +
	`"fd":{"active":true,"priority":5,"storePaths":["` + fdPath + `"]},` +
	`"hello":{"active":true,"attrPath":"legacyPackages.aarch64-darwin.hello","originalUrl":"flake:nixpkgs","outputs":null,"priority":5,"storePaths":["` + helloPath + `"],"url":"` + nixpkgsURL + `"},` +
	`"jq":{"active":true,"attrPath":"legacyPackages.aarch64-darwin.jq","originalUrl":"flake:nixpkgs","outputs":["bin","man"],"priority":5,"storePaths":["` + jqBinPath + `","` + jqManPath + `"],"url":"` + nixpkgsURL + `"}` +
	`},"version":3}`

// manifestV2 is manifestV3 in the version 2 format (elements in an array).
const manifestV2 = `{"version":2,"elements":[` +
	`{"active":true,"priority":5,"storePaths":["` + fdPath + `"]},` +
	`{"active":true,"attrPath":"legacyPackages.aarch64-darwin.hello","originalUrl":"flake:nixpkgs","outputs":null,"priority":5,"storePaths":["` + helloPath + `"],"url":"` + nixpkgsURL + `"}` +
	`]}`

// manifestV1 is manifestV2 in the version 1 format (uri instead of url).
const manifestV1 = `{"version":1,"elements":[` +
	`{"active":true,"priority":5,"storePaths":["` + fdPath + `"]},` +
	`{"active":true,"attrPath":"legacyPackages.aarch64-darwin.hello","originalUri":"flake:nixpkgs","outputs":null,"priority":5,"storePaths":["` + helloPath + `"],"uri":"` + nixpkgsURL + `"}` +
	`]}`

var manifestV3Items = []*NixProfileListItem{
	{
		name:          "fd",
		nixStorePaths: []string{fdPath},
	},
	{
		name:              "hello",
		unlockedReference: "flake:nixpkgs#legacyPackages.aarch64-darwin.hello",
		lockedReference:   nixpkgsURL + "#legacyPackages.aarch64-darwin.hello",
		nixStorePaths:     []string{helloPath},
	},
	{
		name:              "jq",
		unlockedReference: "flake:nixpkgs#legacyPackages.aarch64-darwin.jq",
		lockedReference:   nixpkgsURL + "#legacyPackages.aarch64-darwin.jq",
		nixStorePaths:     []string{jqManPath, jqBinPath},
	},
}

func TestParseManifest(t *testing.T) {
	tests := map[string]struct {
		manifest string
		want     []*NixProfileListItem
	}{
		"v3": {
			manifest: manifestV3,
			want:     manifestV3Items,
		},
		"v3 empty": {
			manifest: `{"elements":{},"version":3}`,
			want:     []*NixProfileListItem{},
		},
		"v3 no url": {
			// nix drops the source when url is empty, and treats missing
			// storePaths as empty.
			manifest: `{"elements":{"x":{"active":false,"attrPath":"a","originalUrl":"flake:nixpkgs","url":""}},"version":3}`,
			want:     []*NixProfileListItem{{name: "x", nixStorePaths: []string{}}},
		},
		"v3 duplicate store paths": {
			manifest: `{"elements":{"fd":{"active":true,"storePaths":["` + fdPath + `","` + fdPath + `"]}},"version":3}`,
			want:     []*NixProfileListItem{{name: "fd", nixStorePaths: []string{fdPath}}},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := parseManifest([]byte(test.manifest))
			if !ok {
				t.Fatal("got ok = false, want true")
			}
			assertItemsEqual(t, got, test.want)
		})
	}
}

func TestParseManifestUnsupported(t *testing.T) {
	tests := map[string]string{
		"v1":                 manifestV1,
		"v2":                 manifestV2,
		"v4":                 `{"elements":{},"version":4}`,
		"no version":         `{"elements":{}}`,
		"v3 elements array":  `{"elements":[],"version":3}`,
		"v3 missing active":  `{"elements":{"fd":{"storePaths":["` + fdPath + `"]}},"version":3}`,
		"v3 wrong type":      `{"elements":{"fd":{"active":true,"priority":"5"}},"version":3}`,
		"invalid json":       `{"elements":`,
		"not an object":      `[]`,
		"version not number": `{"elements":{},"version":"3"}`,
	}
	for name, manifest := range tests {
		t.Run(name, func(t *testing.T) {
			if _, ok := parseManifest([]byte(manifest)); ok {
				t.Error("got ok = true, want false")
			}
		})
	}
}

func TestProfileListItemsReadsManifest(t *testing.T) {
	stubProfileListCommand(t, false)

	// Lay the profile out like nix does: profile -> profile-1-link -> generation.
	dir := t.TempDir()
	gen := filepath.Join(dir, "generation")
	writeManifest(t, gen, "manifest.json", manifestV3)
	if err := os.Symlink(gen, filepath.Join(dir, "profile-1-link")); err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(dir, "profile")
	if err := os.Symlink("profile-1-link", profile); err != nil {
		t.Fatal(err)
	}

	got, err := ProfileListItems(io.Discard, profile)
	if err != nil {
		t.Fatal(err)
	}
	assertItemsEqual(t, got, manifestV3Items)
}

func TestProfileListItemsNoManifest(t *testing.T) {
	stubProfileListCommand(t, false)

	dir := t.TempDir()
	dangling := filepath.Join(dir, "dangling")
	if err := os.Symlink(filepath.Join(dir, "garbage-collected"), dangling); err != nil {
		t.Fatal(err)
	}
	profiles := map[string]string{
		"missing":       filepath.Join(dir, "missing"),
		"dangling link": dangling,
		"empty dir":     t.TempDir(),
	}
	for name, profile := range profiles {
		t.Run(name, func(t *testing.T) {
			got, err := ProfileListItems(io.Discard, profile)
			if err != nil {
				t.Fatal(err)
			}
			if got == nil || len(got) != 0 {
				t.Errorf("got items %v, want empty", got)
			}
		})
	}
}

func TestProfileListItemsFallsBackToCommand(t *testing.T) {
	tests := map[string]struct {
		file     string
		manifest string
	}{
		"v1":         {"manifest.json", manifestV1},
		"v2":         {"manifest.json", manifestV2},
		"v4":         {"manifest.json", `{"elements":{},"version":4}`},
		"bad json":   {"manifest.json", `{`},
		"nix-env":    {"manifest.nix", `[ ]`},
		"unreadable": {"manifest.json/oops", `{}`},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stubProfileListCommand(t, true)
			profile := t.TempDir()
			writeManifest(t, profile, test.file, test.manifest)

			got, err := ProfileListItems(io.Discard, profile)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].name != "from-command" {
				t.Errorf("got items %v, want the stubbed command's items", got)
			}
		})
	}
}

// TestManifestMatchesNixProfileList checks that reading manifest.json gives
// the same items as `nix profile list --json` for the same manifest. nix only
// parses the store paths in the manifest, so they don't need to exist.
func TestManifestMatchesNixProfileList(t *testing.T) {
	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix not found")
	}
	if !nix.AtLeast(nix.Version2_20) {
		t.Skip("version 3 manifests need nix >= 2.20")
	}

	tests := map[string]string{
		"v3":       manifestV3,
		"v3 empty": `{"elements":{},"version":3}`,
		"v3 no url": `{"elements":{"x":{"active":false,"attrPath":"a",` +
			`"originalUrl":"flake:nixpkgs","url":""}},"version":3}`,
		"v3 duplicate store paths": `{"elements":{"fd":{"active":true,` +
			`"storePaths":["` + fdPath + `","` + fdPath + `"]}},"version":3}`,
		"no manifest": "",
	}
	for name, manifest := range tests {
		t.Run(name, func(t *testing.T) {
			profile := t.TempDir()
			if manifest != "" {
				writeManifest(t, profile, "manifest.json", manifest)
			}

			fromFile, ok := readManifestItems(profile)
			if !ok {
				t.Fatal("readManifestItems returned ok = false")
			}
			fromCommand, err := profileListItemsFromCommand(io.Discard, profile)
			if err != nil {
				t.Fatal(err)
			}
			assertItemsEqual(t, fromFile, fromCommand)
		})
	}
}

// stubProfileListCommand replaces the `nix profile list` fallback for the
// duration of the test. If allow is false, the test fails if it's called.
func stubProfileListCommand(t *testing.T, allow bool) {
	t.Helper()
	orig := profileListCommand
	t.Cleanup(func() { profileListCommand = orig })
	profileListCommand = func(io.Writer, string) ([]*NixProfileListItem, error) {
		if !allow {
			t.Error("ProfileListItems ran nix profile list, want it to read manifest.json")
		}
		return []*NixProfileListItem{{name: "from-command"}}, nil
	}
}

func writeManifest(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertItemsEqual(t *testing.T, got, want []*NixProfileListItem) {
	t.Helper()
	byName := func(a, b *NixProfileListItem) int {
		return cmp.Compare(a.NameOrIndex(), b.NameOrIndex())
	}
	got = slices.Clone(got)
	want = slices.Clone(want)
	slices.SortFunc(got, byName)
	slices.SortFunc(want, byName)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got items:\n%v\nwant:\n%v", got, want)
	}
}
