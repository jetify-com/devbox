package shellgen

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/devconfig/configfile"
	"go.jetify.com/devbox/internal/devpkg"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/nix/flake"
)

type lockMock struct{}

func (l *lockMock) Get(key string) *lock.Package {
	return nil
}

func (l *lockMock) Stdenv() flake.Ref {
	return flake.Ref{}
}

func (l *lockMock) ProjectDir() string {
	return ""
}

func (l *lockMock) Resolve(key string) (*lock.Package, error) {
	return &lock.Package{
		Resolved: "github:NixOS/nixpkgs/10b813040df67c4039086db0f6eaf65c536886c6#python312",
	}, nil
}

func TestNewGlibcPatchFlake(t *testing.T) {
	stdenv := flake.Ref{
		Type: flake.TypeGitHub,
		URL:  "https://github.com/NixOS/nixpkgs",
		Ref:  "nixpkgs-unstable",
	}

	packages := devpkg.PackagesFromStringsWithOptions([]string{"python@latest"}, &lockMock{}, devopt.AddOpts{
		Patch: string(configfile.PatchAlways),
	})

	patchFlake, err := newGlibcPatchFlake(stdenv, packages)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if patchFlake.NixpkgsGlibcFlakeRef != stdenv.String() {
		t.Errorf("expected NixpkgsGlibcFlakeRef to be %s, got %s", stdenv.String(), patchFlake.NixpkgsGlibcFlakeRef)
	}

	if len(patchFlake.Outputs.Packages) != 1 {
		t.Errorf("expected 1 package in Outputs, got %d", len(patchFlake.Outputs.Packages))
	}
}

func TestCopySystemCUDALibRemovesStaleLibs(t *testing.T) {
	flakeDir := t.TempDir()
	libDir := filepath.Join(flakeDir, "lib")
	if err := os.Mkdir(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(libDir, "libcuda.so.0.0.0-stale")
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The result depends on whether the system has a CUDA driver, but
	// either way the stale library should be gone.
	_ = (&glibcPatchFlake{}).copySystemCUDALib(flakeDir)
	if _, err := os.Lstat(stale); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("got stale library error %v, want %v", err, fs.ErrNotExist)
	}
}
