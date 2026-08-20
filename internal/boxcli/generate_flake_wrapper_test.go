// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.jetify.com/devbox/internal/devbox/flakegen"
)

// runFlakeWrapper invokes the flake-wrapper subcommand in-process and returns
// its stdout and error. It isolates HOME and the working directory so the
// command's devbox.Open fallback does not touch the user's real projects.
func runFlakeWrapper(t *testing.T, args ...string) (string, error) {
	t.Helper()
	// Keep devbox.Open from picking up state outside the sandbox.
	t.Setenv("HOME", t.TempDir())

	cmd := genFlakeWrapperCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}

func writeNixFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(
		path,
		[]byte("{ stdenv }: stdenv.mkDerivation { name = \"noop\"; }\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
}

func writeDefaultNix(t *testing.T, dir string) {
	t.Helper()
	writeNixFile(t, filepath.Join(dir, "default.nix"))
}

func TestGenFlakeWrapper_WritesFlakeForDirectory(t *testing.T) {
	dir := t.TempDir()
	writeDefaultNix(t, dir)

	stdout, err := runFlakeWrapper(t, dir)
	if err != nil {
		t.Fatalf("flake-wrapper returned error: %v", err)
	}

	flakePath := filepath.Join(dir, "flake.nix")
	data, err := os.ReadFile(flakePath)
	if err != nil {
		t.Fatalf("expected flake.nix to be written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "pkgs.callPackage ./default.nix {}") {
		t.Errorf("flake.nix missing callPackage line:\n%s", content)
	}
	if !strings.Contains(content, flakegen.DefaultNixpkgsURL) {
		t.Errorf(
			"expected default nixpkgs URL %q in flake.nix:\n%s",
			flakegen.DefaultNixpkgsURL, content,
		)
	}
	if !strings.Contains(content, "packages = forAllSystems") {
		t.Errorf("flake.nix missing forAllSystems block:\n%s", content)
	}
	if !strings.Contains(stdout, flakePath) {
		t.Errorf("stdout summary should reference %s, got:\n%s", flakePath, stdout)
	}
}

func TestGenFlakeWrapper_RefusesToOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	writeDefaultNix(t, dir)

	if _, err := runFlakeWrapper(t, dir); err != nil {
		t.Fatalf("first invocation failed: %v", err)
	}

	flakePath := filepath.Join(dir, "flake.nix")
	sentinel := "# user edit\n"
	if err := os.WriteFile(flakePath, []byte(sentinel), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runFlakeWrapper(t, dir); err == nil {
		t.Fatal("expected error when flake.nix already exists without --force")
	}

	data, err := os.ReadFile(flakePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != sentinel {
		t.Errorf("flake.nix should be unchanged without --force, got:\n%s", data)
	}

	if _, err := runFlakeWrapper(t, dir, "--force"); err != nil {
		t.Fatalf("unexpected error with --force: %v", err)
	}
	data, err = os.ReadFile(flakePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == sentinel {
		t.Errorf("--force should have overwritten the user edit")
	}
	if !strings.Contains(string(data), "pkgs.callPackage ./default.nix {}") {
		t.Errorf("overwritten flake.nix missing callPackage line:\n%s", data)
	}
}

func TestGenFlakeWrapper_PrintDoesNotWriteFile(t *testing.T) {
	dir := t.TempDir()
	writeDefaultNix(t, dir)

	stdout, err := runFlakeWrapper(t, dir, "--print")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "pkgs.callPackage ./default.nix {}") {
		t.Errorf("--print should emit the rendered template, got:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, "flake.nix")); !os.IsNotExist(err) {
		t.Errorf("--print should not create flake.nix, stat err=%v", err)
	}
}

func TestGenFlakeWrapper_AcceptsDefaultNixFilePath(t *testing.T) {
	dir := t.TempDir()
	writeDefaultNix(t, dir)

	if _, err := runFlakeWrapper(t, filepath.Join(dir, "default.nix")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "flake.nix")); err != nil {
		t.Errorf("expected flake.nix to be written in parent dir: %v", err)
	}
}

func TestGenFlakeWrapper_AcceptsNamedNixFilePath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "my_package.nix")
	writeNixFile(t, target)

	stdout, err := runFlakeWrapper(t, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	flakePath := filepath.Join(dir, "flake.nix")
	data, err := os.ReadFile(flakePath)
	if err != nil {
		t.Fatalf("expected flake.nix to be written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "pkgs.callPackage ./my_package.nix {}") {
		t.Errorf("flake.nix should callPackage the named file:\n%s", content)
	}
	if strings.Contains(content, "./default.nix") {
		t.Errorf("flake.nix should not reference default.nix:\n%s", content)
	}
	if !strings.Contains(stdout, flakePath) {
		t.Errorf("stdout should reference %s, got:\n%s", flakePath, stdout)
	}
}

func TestGenFlakeWrapper_RejectsNonNixFile(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "other.txt")
	if err := os.WriteFile(other, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := runFlakeWrapper(t, other)
	if err == nil {
		t.Fatal("expected error when pointed at a non-.nix file")
	}
	if !strings.Contains(err.Error(), ".nix") {
		t.Errorf("error should mention .nix extension, got: %v", err)
	}
}

func TestGenFlakeWrapper_MissingDefaultNixFails(t *testing.T) {
	dir := t.TempDir()

	_, err := runFlakeWrapper(t, dir)
	if err == nil {
		t.Fatal("expected error when directory has no default.nix")
	}
	if !strings.Contains(err.Error(), "default.nix") {
		t.Errorf("error should mention default.nix, got: %v", err)
	}
}

func TestGenFlakeWrapper_NixpkgsOverride(t *testing.T) {
	dir := t.TempDir()
	writeDefaultNix(t, dir)

	const customURL = "github:NixOS/nixpkgs/nixos-23.11"
	if _, err := runFlakeWrapper(t, dir, "--nixpkgs", customURL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "flake.nix"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), customURL) {
		t.Errorf("expected custom nixpkgs URL %q in flake.nix:\n%s", customURL, data)
	}
	if strings.Contains(string(data), flakegen.DefaultNixpkgsURL) {
		t.Errorf(
			"default nixpkgs URL should not appear when --nixpkgs is set:\n%s",
			data,
		)
	}
}

func TestGenFlakeWrapper_AttrOverride(t *testing.T) {
	dir := t.TempDir()
	writeDefaultNix(t, dir)

	if _, err := runFlakeWrapper(t, dir, "--attr", "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "flake.nix"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "hello = pkgs.callPackage ./default.nix {}") {
		t.Errorf("expected custom attr in flake.nix:\n%s", data)
	}
}
