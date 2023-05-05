// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

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
			Definitions []string
			DevPackages []string
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
	Definitions map[string]string
	DevPackages []string
	FlakeInputs []plansdk.FlakeInput
}{
	NixpkgsInfo: struct {
		URL string
	}{
		URL: "https://github.com/nixos/nixpkgs/archive/b9c00c1d41ccd6385da243415299b39aa73357be.tar.gz",
	},
	Definitions: map[string]string{
		"php":                    "pkgs.php.withExtensions ({ enabled, all }: enabled ++ (with all; [ blackfire ]));",
		"php81Packages.composer": "php.packages.composer;",
	},
	DevPackages: []string{
		"php",
		"php81Packages.composer",
		"php81Extensions.blackfire",
		"flyctl",
		"postgresql",
		"tree",
		"git",
		"zsh",
		"openssh",
		"vim",
		"sqlite",
		"jq",
		"delve",
		"ripgrep",
		"shellcheck",
		"terraform",
		"xz",
		"zstd",
		"gnupg",
		"go_1_20",
		"python3",
		"graphviz",
	},
}
