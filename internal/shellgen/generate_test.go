// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package shellgen

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/searcher"
)

// update overwrites golden files with the new test results.
var update = flag.Bool("update", false, "update the golden files with the test results")

// TestWriteFromTemplate will verify that the flake.nix code generation works as expected.
// Note: this test was derived from an older flake.nix, prior to having the builtins.FetchClosures
// and so may be a bit out of date. It could be updated to be better and more exhaustive.
func TestWriteFromTemplate(t *testing.T) {
	t.Setenv("__DEVBOX_NIX_SYSTEM", "x86_64-linux")
	dir := filepath.Join(t.TempDir(), "makeme")
	outPath := filepath.Join(dir, "flake.nix")
	err := writeFromTemplate(dir, testFlakeTmplPlan, "flake.nix", "flake.nix")
	if err != nil {
		t.Fatal("got error writing flake template:", err)
	}
	cmpGoldenFile(t, outPath, "testdata/flake.nix.golden")

	t.Run("WriteUnmodified", func(t *testing.T) {
		err = writeFromTemplate(dir, testFlakeTmplPlan, "flake.nix", "flake.nix")
		if err != nil {
			t.Fatal("got error writing flake template:", err)
		}
		cmpGoldenFile(t, outPath, "testdata/flake.nix.golden")
	})
	t.Run("WriteModifiedSmaller", func(t *testing.T) {
		emptyPlan := &flakePlan{
			NixpkgsInfo: &NixpkgsInfo{
				URL:    "",
				TarURL: "",
			},
			Packages:    []*devpkg.Package{},
			FlakeInputs: []flakeInput{},
			System:      "x86_64-linux",
		}
		err = writeFromTemplate(dir, emptyPlan, "flake.nix", "flake.nix")
		if err != nil {
			t.Fatal("got error writing flake template:", err)
		}
		cmpGoldenFile(t, outPath, "testdata/flake-empty.nix.golden")
	})
}

func cmpGoldenFile(t *testing.T, gotPath, wantGoldenPath string) {
	got, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatal("got error reading generated file:", err)
	}
	if *update {
		err = os.WriteFile(wantGoldenPath, got, 0o666)
		if err != nil {
			t.Error("got error updating golden file:", err)
		}
	}
	golden, err := os.ReadFile(wantGoldenPath)
	if err != nil {
		t.Fatal("got error reading golden file:", err)
	}
	diff := cmp.Diff(golden, got)
	if diff != "" {
		t.Errorf(strings.TrimSpace(`
got wrong file contents (-%s +%s):

%s
If the new file is correct, you can update the golden file with:

	go test -run "^%s$" -update`),
			filepath.Base(wantGoldenPath), filepath.Base(gotPath),
			diff, t.Name())
	}
}

var (
	locker            = &lockmock{}
	testFlakeTmplPlan = &flakePlan{
		NixpkgsInfo: &NixpkgsInfo{
			URL:    "https://github.com/nixos/nixpkgs/archive/b9c00c1d41ccd6385da243415299b39aa73357be.tar.gz",
			TarURL: "", // TODO savil
		},
		Packages: []*devpkg.Package{}, // TODO savil
		FlakeInputs: []flakeInput{
			{
				Name: "nixpkgs",
				URL:  "github:NixOS/nixpkgs/b9c00c1d41ccd6385da243415299b39aa73357be",
				Packages: []*devpkg.Package{
					devpkg.PackageFromStringWithDefaults("php@latest", locker),
					devpkg.PackageFromStringWithDefaults("php81Packages.composer@latest", locker),
					devpkg.PackageFromStringWithDefaults("php81Extensions.blackfire@latest", locker),
					devpkg.PackageFromStringWithDefaults("flyctl@latest", locker),
					devpkg.PackageFromStringWithDefaults("postgresql@latest", locker),
					devpkg.PackageFromStringWithDefaults("tree@latest", locker),
					devpkg.PackageFromStringWithDefaults("git@latest", locker),
					devpkg.PackageFromStringWithDefaults("zsh@latest", locker),
					devpkg.PackageFromStringWithDefaults("openssh@latest", locker),
					devpkg.PackageFromStringWithDefaults("vim@latest", locker),
					devpkg.PackageFromStringWithDefaults("sqlite@latest", locker),
					devpkg.PackageFromStringWithDefaults("jq@latest", locker),
					devpkg.PackageFromStringWithDefaults("delve@latest", locker),
					devpkg.PackageFromStringWithDefaults("ripgrep@latest", locker),
					devpkg.PackageFromStringWithDefaults("shellcheck@latest", locker),
					devpkg.PackageFromStringWithDefaults("terraform@latest", locker),
					devpkg.PackageFromStringWithDefaults("xz@latest", locker),
					devpkg.PackageFromStringWithDefaults("zstd@latest", locker),
					devpkg.PackageFromStringWithDefaults("gnupg@latest", locker),
					devpkg.PackageFromStringWithDefaults("go_1_20@latest", locker),
					devpkg.PackageFromStringWithDefaults("python3@latest", locker),
					devpkg.PackageFromStringWithDefaults("graphviz@latest", locker),
				},
			},
		},
		System: "x86_64-linux",
	}
)

type lockmock struct{}

func (*lockmock) Resolve(pkg string) (*lock.Package, error) {
	name, _, _ := searcher.ParseVersionedPackage(pkg)
	return &lock.Package{
		Resolved: "github:NixOS/nixpkgs/b22db301217578a8edfccccf5cedafe5fc54e78b#" + name,
	}, nil
}

func (*lockmock) Get(pkg string) *lock.Package {
	return nil
}

func (*lockmock) LegacyNixpkgsPath(pkg string) string {
	return ""
}

func (*lockmock) ProjectDir() string {
	return ""
}
