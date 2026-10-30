// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devpkg

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/samber/lo"
	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/nix/flake"
)

const nixCommitHash = "hsdafkhsdafhas"

type inputTestCase struct {
	pkg                string
	isFlake            bool
	name               string
	urlWithoutFragment string
	urlForInput        string
}

func TestInput(t *testing.T) {
	projectDir := "/tmp/my-project"
	cases := []inputTestCase{
		{
			pkg:                "path:path/to/my-flake#my-package",
			isFlake:            true,
			name:               "my-flake-9a897d",
			urlWithoutFragment: "path:" + filepath.Join(projectDir, "path/to/my-flake"),
			urlForInput:        "path:" + filepath.Join(projectDir, "path/to/my-flake"),
		},
		{
			pkg:                "path:.#my-package",
			isFlake:            true,
			name:               "my-project-45b022",
			urlWithoutFragment: "path:" + projectDir,
			urlForInput:        "path:" + projectDir,
		},
		{
			pkg:                "path:/tmp/my-project/path/to/my-flake#my-package",
			isFlake:            true,
			name:               "my-flake-9a897d",
			urlWithoutFragment: "path:" + filepath.Join(projectDir, "path/to/my-flake"),
			urlForInput:        "path:" + filepath.Join(projectDir, "path/to/my-flake"),
		},
		{
			pkg:                "path:/tmp/my-project/path/to/my-flake",
			isFlake:            true,
			name:               "my-flake-7d03be",
			urlWithoutFragment: "path:" + filepath.Join(projectDir, "path/to/my-flake"),
			urlForInput:        "path:" + filepath.Join(projectDir, "path/to/my-flake"),
		},
		{
			pkg:                "hello",
			isFlake:            false,
			name:               "nixpkgs-hsdafk",
			urlWithoutFragment: "hello",
			urlForInput:        fmt.Sprintf("github:NixOS/nixpkgs/%s", nixCommitHash),
		},
		{
			pkg:                "hello@123",
			isFlake:            false,
			name:               "nixpkgs-hsdafk",
			urlWithoutFragment: "hello@123",
			urlForInput:        fmt.Sprintf("github:NixOS/nixpkgs/%s", nixCommitHash),
		},
		{
			pkg:                "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello",
			isFlake:            true,
			name:               "gh-nixos-nixpkgs-5233fd2ba76a3accb5aaa999c00509a11fd0793c",
			urlWithoutFragment: "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c",
			urlForInput:        "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c",
		},
		{
			pkg:                "github:F1bonacc1/process-compose",
			isFlake:            true,
			name:               "gh-F1bonacc1-process-compose",
			urlWithoutFragment: "github:F1bonacc1/process-compose",
			urlForInput:        "github:F1bonacc1/process-compose",
		},
	}

	for _, testCase := range cases {
		i := testInputFromString(testCase.pkg, projectDir)
		if name := i.FlakeInputName(); testCase.name != name {
			t.Errorf("Name() = %v, want %v", name, testCase.name)
		}
		if urlForInput := i.URLForFlakeInput(); testCase.urlForInput != urlForInput {
			t.Errorf("URLForFlakeInput() = %v, want %v", urlForInput, testCase.urlForInput)
		}
	}
}

func TestFlakeInputNameUnique(t *testing.T) {
	pkgs := []string{
		"github:owner/repo/abc123#x",
		"github:owner/repo/abc123?dir=a#x",
		"github:owner/repo/abc123?dir=b#y",
		"github:owner/repo/abc123?host=git.example.com#x",
	}
	seen := map[string]string{}
	for _, pkg := range pkgs {
		name := testInputFromString(pkg, "/tmp/my-project").FlakeInputName()
		if other, ok := seen[name]; ok {
			t.Errorf("FlakeInputName() = %q for both %q and %q", name, other, pkg)
		}
		seen[name] = pkg
	}
}

type testInput struct {
	*Package
}

type lockfile struct {
	projectDir string
}

func (l *lockfile) ProjectDir() string {
	return l.projectDir
}

func (l *lockfile) Stdenv() flake.Ref {
	return flake.Ref{
		Type:  flake.TypeGitHub,
		Owner: "NixOS",
		Repo:  "nixpkgs",
		Rev:   nixCommitHash,
	}
}

func (l *lockfile) Get(pkg string) *lock.Package {
	return nil
}

func (l *lockfile) Resolve(pkg string) (*lock.Package, error) {
	switch {
	case strings.Contains(pkg, "path:"):
		return &lock.Package{Resolved: pkg}, nil
	case strings.Contains(pkg, "github:"):
		return &lock.Package{Resolved: pkg}, nil
	default:
		return &lock.Package{
			Resolved: flake.Installable{
				Ref:      l.Stdenv(),
				AttrPath: pkg,
			}.String(),
		}, nil
	}
}

func testInputFromString(s, projectDir string) *testInput {
	return lo.ToPtr(testInput{Package: PackageFromStringWithDefaults(s, &lockfile{projectDir})})
}

func TestHashFromNixPkgsURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{
			url:      "github:NixOS/nixpkgs/12345",
			expected: "12345",
		},
		{
			url:      "github:NixOS/nixpkgs/abcdef#hello",
			expected: "abcdef",
		},
		{
			url:      "github:NixOS/nixpkgs/",
			expected: "",
		},
		{
			url:      "github:NixOS/nixpkgs",
			expected: "",
		},
		{
			url:      "github:NixOS/other-repo/12345",
			expected: "",
		},
		{
			url:      "",
			expected: "",
		},
	}

	for _, test := range tests {
		result := nix.HashFromNixPkgsURL(test.url)
		if result != test.expected {
			t.Errorf(
				"Expected hash '%s' for URL '%s', but got '%s'",
				test.expected,
				test.url,
				result,
			)
		}
	}
}

func TestCanonicalName(t *testing.T) {
	tests := []struct {
		pkgName      string
		expectedName string
	}{
		{"go", "go"},
		{"go@latest", "go"},
		{"go@1.21", "go"},
		{"nodePackages.@vue/cli@5.0", "nodePackages.@vue/cli"},
		{"emacsPackages.@", "emacsPackages.@"},
		{"runx:golangci/golangci-lint@latest", "runx:golangci/golangci-lint"},
		{"runx:golangci/golangci-lint@v0.0.2", "runx:golangci/golangci-lint"},
		{"runx:golangci/golangci-lint", "runx:golangci/golangci-lint"},
		{"pkg:./go.pkg.json@1.22", "pkg:./go.pkg.json"},
		{"pkg:./go.pkg.json", "pkg:./go.pkg.json"},
		{"pkg:./go.pkg.jsonc@1.22", "pkg:./go.pkg.jsonc"},
		{"pkg:https://cdn.example.com/@acme/tool.pkg.json@latest", "pkg:https://cdn.example.com/@acme/tool.pkg.json"},
		{"github:NixOS/nixpkgs/12345", ""},
		{"path:/to/my/file", ""},
	}

	for _, tt := range tests {
		t.Run(tt.pkgName, func(t *testing.T) {
			pkg := PackageFromStringWithDefaults(tt.pkgName, &lockfile{})
			got := pkg.CanonicalName()
			if got != tt.expectedName {
				t.Errorf("Expected canonical name %q, but got %q", tt.expectedName, got)
			}
		})
	}
}

func TestVersioned(t *testing.T) {
	tests := []struct {
		pkgName   string
		version   string
		versioned string
	}{
		{"go", "", "go@latest"},
		{"go@1.21", "1.21", "go@1.21"},
		{"nodePackages.@vue/cli@5.0", "5.0", "nodePackages.@vue/cli@5.0"},
	}
	for _, tt := range tests {
		t.Run(tt.pkgName, func(t *testing.T) {
			pkg := PackageFromStringWithDefaults(tt.pkgName, &lockfile{})
			if got := pkg.version(); got != tt.version {
				t.Errorf("version() = %q, want %q", got, tt.version)
			}
			if got := pkg.Versioned(); got != tt.versioned {
				t.Errorf("Versioned() = %q, want %q", got, tt.versioned)
			}
		})
	}
}

func TestFlakeOutputNames(t *testing.T) {
	tests := []struct {
		pkg     string
		opts    []string
		outputs []string
	}{
		{"github:NixOS/nixpkgs/12345#curl", nil, []string{}},
		{"github:NixOS/nixpkgs/12345#curl", []string{"dev", "out"}, []string{"dev", "out"}},
		{"github:NixOS/nixpkgs/12345#curl^bin,dev", nil, []string{"bin", "dev"}},
		{"github:NixOS/nixpkgs/12345#curl^bin", []string{"dev"}, []string{"bin", "dev"}},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			pkg := PackageFromStringWithOptions(tt.pkg, &lockfile{}, devopt.AddOpts{Outputs: tt.opts})
			got := pkg.outputs.selectedNames
			if !slices.Equal(got, tt.outputs) {
				t.Errorf("got outputs %q, want %q", got, tt.outputs)
			}
		})
	}
}
