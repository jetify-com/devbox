// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
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
		emptyPlan := struct {
			NixpkgsInfo struct {
				URL string
			}
			FlakeInputs []flakeInput
		}{}
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
	testFlakeTmplPlan = &struct {
		NixpkgsInfo struct {
			URL string
		}
		FlakeInputs []flakeInput
	}{
		NixpkgsInfo: struct {
			URL string
		}{
			URL: "https://github.com/nixos/nixpkgs/archive/b9c00c1d41ccd6385da243415299b39aa73357be.tar.gz",
		},
		FlakeInputs: []flakeInput{
			{
				Name: "nixpkgs",
				URL:  "github:NixOS/nixpkgs/b9c00c1d41ccd6385da243415299b39aa73357be",
				Packages: []*devpkg.Package{
					devpkg.PackageFromString("php@latest", locker),
					devpkg.PackageFromString("php81Packages.composer@latest", locker),
					devpkg.PackageFromString("php81Extensions.blackfire@latest", locker),
					devpkg.PackageFromString("flyctl@latest", locker),
					devpkg.PackageFromString("postgresql@latest", locker),
					devpkg.PackageFromString("tree@latest", locker),
					devpkg.PackageFromString("git@latest", locker),
					devpkg.PackageFromString("zsh@latest", locker),
					devpkg.PackageFromString("openssh@latest", locker),
					devpkg.PackageFromString("vim@latest", locker),
					devpkg.PackageFromString("sqlite@latest", locker),
					devpkg.PackageFromString("jq@latest", locker),
					devpkg.PackageFromString("delve@latest", locker),
					devpkg.PackageFromString("ripgrep@latest", locker),
					devpkg.PackageFromString("shellcheck@latest", locker),
					devpkg.PackageFromString("terraform@latest", locker),
					devpkg.PackageFromString("xz@latest", locker),
					devpkg.PackageFromString("zstd@latest", locker),
					devpkg.PackageFromString("gnupg@latest", locker),
					devpkg.PackageFromString("go_1_20@latest", locker),
					devpkg.PackageFromString("python3@latest", locker),
					devpkg.PackageFromString("graphviz@latest", locker),
				},
			},
		},
	}
)

type lockmock struct{}

func (*lockmock) Resolve(pkg string) (*lock.Package, error) {
	name, _, _ := searcher.ParseVersionedPackage(pkg)
	return &lock.Package{
		Resolved: "#" + name,
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
