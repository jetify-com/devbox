// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devpkg

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
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

type testInput struct {
	*Package
}

type lockfile struct {
	projectDir string
}

func (l *lockfile) ProjectDir() string {
	return l.projectDir
}

func (l *lockfile) LegacyNixpkgsPath(pkg string) string {
	return fmt.Sprintf(
		"github:NixOS/nixpkgs/%s#%s",
		nixCommitHash,
		pkg,
	)
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
			Resolved: l.LegacyNixpkgsPath(pkg),
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
		{"runx:golangci/golangci-lint@latest", "runx:golangci/golangci-lint"},
		{"runx:golangci/golangci-lint@v0.0.2", "runx:golangci/golangci-lint"},
		{"runx:golangci/golangci-lint", "runx:golangci/golangci-lint"},
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
