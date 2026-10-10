// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package flakegen_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.jetify.com/devbox/internal/devbox/flakegen"
)

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

func writeDefaultNix(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "default.nix")
	writeNixFile(t, p)
	return p
}

// evalSymlinks resolves symlinks so tests can compare paths on systems (like
// macOS) where t.TempDir lives under /var but resolves to /private/var.
func evalSymlinks(t *testing.T, p string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestResolveNixFile_Directory(t *testing.T) {
	dir := t.TempDir()
	writeDefaultNix(t, dir)

	got, err := flakegen.ResolveNixFile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "default.nix")
	if evalSymlinks(t, got) != evalSymlinks(t, want) {
		t.Errorf("ResolveNixFile(%q) = %q, want %q", dir, got, want)
	}
}

func TestResolveNixFile_DirectoryMissingDefaultNix(t *testing.T) {
	dir := t.TempDir()

	_, err := flakegen.ResolveNixFile(dir)
	if err == nil {
		t.Fatal("expected error when directory has no default.nix")
	}
	if !strings.Contains(err.Error(), "default.nix") {
		t.Errorf("error should mention default.nix, got: %v", err)
	}
}

func TestResolveNixFile_DefaultNixFile(t *testing.T) {
	dir := t.TempDir()
	target := writeDefaultNix(t, dir)

	got, err := flakegen.ResolveNixFile(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evalSymlinks(t, got) != evalSymlinks(t, target) {
		t.Errorf("ResolveNixFile(%q) = %q, want %q", target, got, target)
	}
}

func TestResolveNixFile_NamedNixFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "my_package.nix")
	writeNixFile(t, target)

	got, err := flakegen.ResolveNixFile(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evalSymlinks(t, got) != evalSymlinks(t, target) {
		t.Errorf("ResolveNixFile(%q) = %q, want %q", target, got, target)
	}
}

func TestResolveNixFile_NonNixFile(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "other.txt")
	if err := os.WriteFile(other, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := flakegen.ResolveNixFile(other)
	if err == nil {
		t.Fatal("expected error for non-.nix file")
	}
	if !strings.Contains(err.Error(), ".nix") {
		t.Errorf("error should mention .nix extension, got: %v", err)
	}
}

func TestResolveNixFile_Missing(t *testing.T) {
	_, err := flakegen.ResolveNixFile(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestGenerate_WritesFlake(t *testing.T) {
	dir := t.TempDir()
	nixFile := writeDefaultNix(t, dir)

	flakePath, err := flakegen.Generate(flakegen.Opts{NixFile: nixFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flakePath != filepath.Join(dir, "flake.nix") {
		t.Errorf("unexpected flakePath %q", flakePath)
	}
	data, err := os.ReadFile(flakePath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		"pkgs.callPackage ./default.nix {}",
		flakegen.DefaultNixpkgsURL,
		"default = pkgs.callPackage",
		"packages = forAllSystems",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("flake.nix missing %q:\n%s", want, content)
		}
	}
}

func TestGenerate_WritesFlakeForNamedNixFile(t *testing.T) {
	dir := t.TempDir()
	nixFile := filepath.Join(dir, "my_package.nix")
	writeNixFile(t, nixFile)

	flakePath, err := flakegen.Generate(flakegen.Opts{NixFile: nixFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(flakePath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "pkgs.callPackage ./my_package.nix {}") {
		t.Errorf("flake.nix should callPackage the custom file:\n%s", content)
	}
	if strings.Contains(content, "./default.nix") {
		t.Errorf("flake.nix should not reference default.nix:\n%s", content)
	}
}

func TestGenerate_DefaultsAttrAndNixpkgs(t *testing.T) {
	dir := t.TempDir()
	nixFile := writeDefaultNix(t, dir)

	if _, err := flakegen.Generate(flakegen.Opts{
		NixFile:    nixFile,
		NixpkgsURL: "",
		Attr:       "",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "flake.nix"))
	if !strings.Contains(string(data), flakegen.DefaultNixpkgsURL) {
		t.Errorf("empty NixpkgsURL should fall back to DefaultNixpkgsURL:\n%s", data)
	}
	if !strings.Contains(string(data), "default = pkgs.callPackage") {
		t.Errorf("empty Attr should fall back to \"default\":\n%s", data)
	}
}

func TestGenerate_RefusesOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	nixFile := writeDefaultNix(t, dir)

	if _, err := flakegen.Generate(flakegen.Opts{NixFile: nixFile}); err != nil {
		t.Fatal(err)
	}

	sentinel := "# user edit\n"
	flakePath := filepath.Join(dir, "flake.nix")
	if err := os.WriteFile(flakePath, []byte(sentinel), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := flakegen.Generate(flakegen.Opts{NixFile: nixFile}); err == nil {
		t.Fatal("expected error without Force")
	}
	data, _ := os.ReadFile(flakePath)
	if string(data) != sentinel {
		t.Errorf("flake.nix should be unchanged without Force, got:\n%s", data)
	}

	if _, err := flakegen.Generate(flakegen.Opts{NixFile: nixFile, Force: true}); err != nil {
		t.Fatalf("unexpected error with Force: %v", err)
	}
	data, _ = os.ReadFile(flakePath)
	if string(data) == sentinel {
		t.Errorf("Force should have overwritten the user edit")
	}
}

func TestGenerate_Print(t *testing.T) {
	dir := t.TempDir()
	nixFile := writeDefaultNix(t, dir)

	buf := &bytes.Buffer{}
	flakePath, err := flakegen.Generate(flakegen.Opts{
		NixFile: nixFile,
		Print:   true,
		Out:     buf,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flakePath != "" {
		t.Errorf("Print should return empty flakePath, got %q", flakePath)
	}
	if !strings.Contains(buf.String(), "pkgs.callPackage ./default.nix {}") {
		t.Errorf("Print should emit the rendered template, got:\n%s", buf.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "flake.nix")); !os.IsNotExist(err) {
		t.Errorf("Print should not create flake.nix, stat err=%v", err)
	}
}

func TestGenerate_PrintRequiresOut(t *testing.T) {
	dir := t.TempDir()
	nixFile := writeDefaultNix(t, dir)

	_, err := flakegen.Generate(flakegen.Opts{NixFile: nixFile, Print: true})
	if err == nil {
		t.Fatal("expected error when Print is true but Out is nil")
	}
}

func TestGenerate_CustomAttrAndNixpkgs(t *testing.T) {
	dir := t.TempDir()
	nixFile := writeDefaultNix(t, dir)

	const customURL = "github:NixOS/nixpkgs/nixos-23.11"
	if _, err := flakegen.Generate(flakegen.Opts{
		NixFile:    nixFile,
		NixpkgsURL: customURL,
		Attr:       "hello",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "flake.nix"))
	content := string(data)
	if !strings.Contains(content, customURL) {
		t.Errorf("expected custom nixpkgs URL %q:\n%s", customURL, content)
	}
	if strings.Contains(content, flakegen.DefaultNixpkgsURL) {
		t.Errorf("default URL should not appear when NixpkgsURL is set:\n%s", content)
	}
	if !strings.Contains(content, "hello = pkgs.callPackage ./default.nix {}") {
		t.Errorf("expected custom attr in flake.nix:\n%s", content)
	}
}

func TestGenerate_MissingNixFile(t *testing.T) {
	_, err := flakegen.Generate(flakegen.Opts{NixFile: ""})
	if err == nil {
		t.Fatal("expected error when NixFile is empty")
	}
}

func TestGenerate_NixFileDoesNotExist(t *testing.T) {
	dir := t.TempDir()

	_, err := flakegen.Generate(flakegen.Opts{
		NixFile: filepath.Join(dir, "my_package.nix"),
	})
	if err == nil {
		t.Fatal("expected error when the named file does not exist")
	}
	if !strings.Contains(err.Error(), "my_package.nix") {
		t.Errorf("error should mention my_package.nix, got: %v", err)
	}
}

func TestGenerate_NixFileWrongExtension(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "default.txt")
	if err := os.WriteFile(target, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := flakegen.Generate(flakegen.Opts{NixFile: target})
	if err == nil {
		t.Fatal("expected error when NixFile lacks a .nix extension")
	}
	if !strings.Contains(err.Error(), ".nix") {
		t.Errorf("error should mention .nix extension, got: %v", err)
	}
}
