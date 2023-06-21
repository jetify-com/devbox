// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package filegen

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

// update overwrites golden files with the new test results.
var update = flag.Bool("update", false, "update the golden files with the test results")

func TestWriteFromTemplate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "makeme")
	outPath := filepath.Join(dir, "flake.nix")
	err := writeFromTemplate(dir, testFlakeTmplPlan, "flake.nix")
	if err != nil {
		t.Fatal("got error writing flake template:", err)
	}
	cmpGoldenFile(t, outPath, "testdata/flake.nix.golden")

	t.Run("WriteUnmodified", func(t *testing.T) {
		err = writeFromTemplate(dir, testFlakeTmplPlan, "flake.nix")
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
			FlakeInputs []plansdk.FlakeInput
		}{}
		err = writeFromTemplate(dir, emptyPlan, "flake.nix")
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
		err = os.WriteFile(wantGoldenPath, got, 0666)
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

var testFlakeTmplPlan = &struct {
	NixpkgsInfo struct {
		URL string
	}
	FlakeInputs []plansdk.FlakeInput
}{
	NixpkgsInfo: struct {
		URL string
	}{
		URL: "https://github.com/nixos/nixpkgs/archive/b9c00c1d41ccd6385da243415299b39aa73357be.tar.gz",
	},
	FlakeInputs: []plansdk.FlakeInput{
		{
			Name: "nixpkgs",
			URL:  "github:NixOS/nixpkgs/b9c00c1d41ccd6385da243415299b39aa73357be",
			Packages: []string{
				"legacyPackages.aarch64-darwin.php",
				"legacyPackages.aarch64-darwin.php81Packages.composer",
				"legacyPackages.aarch64-darwin.php81Extensions.blackfire",
				"legacyPackages.aarch64-darwin.flyctl",
				"legacyPackages.aarch64-darwin.postgresql",
				"legacyPackages.aarch64-darwin.tree",
				"legacyPackages.aarch64-darwin.git",
				"legacyPackages.aarch64-darwin.zsh",
				"legacyPackages.aarch64-darwin.openssh",
				"legacyPackages.aarch64-darwin.vim",
				"legacyPackages.aarch64-darwin.sqlite",
				"legacyPackages.aarch64-darwin.jq",
				"legacyPackages.aarch64-darwin.delve",
				"legacyPackages.aarch64-darwin.ripgrep",
				"legacyPackages.aarch64-darwin.shellcheck",
				"legacyPackages.aarch64-darwin.terraform",
				"legacyPackages.aarch64-darwin.xz",
				"legacyPackages.aarch64-darwin.zstd",
				"legacyPackages.aarch64-darwin.gnupg",
				"legacyPackages.aarch64-darwin.go_1_20",
				"legacyPackages.aarch64-darwin.python3",
				"legacyPackages.aarch64-darwin.graphviz",
			},
		},
	},
}
