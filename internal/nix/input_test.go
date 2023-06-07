// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/lock"
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
			name:               "my-flake-773986",
			urlWithoutFragment: "path:" + filepath.Join(projectDir, "path/to/my-flake"),
			urlForInput:        "path:" + filepath.Join(projectDir, "path/to/my-flake"),
		},
		{
			pkg:                "path:.#my-package",
			isFlake:            true,
			name:               "my-project-20698c",
			urlWithoutFragment: "path:" + projectDir,
			urlForInput:        "path:" + projectDir,
		},
		{
			pkg:                "path:/tmp/my-project/path/to/my-flake#my-package",
			isFlake:            true,
			name:               "my-flake-773986",
			urlWithoutFragment: "path:" + filepath.Join(projectDir, "path/to/my-flake"),
			urlForInput:        "path:" + filepath.Join(projectDir, "path/to/my-flake"),
		},
		{
			pkg:                "path:/tmp/my-project/path/to/my-flake",
			isFlake:            true,
			name:               "my-flake-eaedce",
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
		if name := i.InputName(); testCase.name != name {
			t.Errorf("Name() = %v, want %v", name, testCase.name)
		}
		if urlWithoutFragment := i.urlWithoutFragment(); testCase.urlWithoutFragment != urlWithoutFragment {
			t.Errorf("URLWithoutFragment() = %v, want %v", urlWithoutFragment, testCase.urlWithoutFragment)
		}
		if urlForInput := i.URLForInput(); testCase.urlForInput != urlForInput {
			t.Errorf("URLForInput() = %v, want %v", urlForInput, testCase.urlForInput)
		}
	}
}

type testInput struct {
	Input
}

type lockfile struct {
	projectDir string
}

func (lockfile) ConfigHash() (string, error) {
	return "", nil
}

func (lockfile) NixPkgsCommitHash() string {
	return ""
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
	return lo.ToPtr(testInput{Input: *InputFromString(s, &lockfile{projectDir})})
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
		result := HashFromNixPkgsURL(test.url)
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
