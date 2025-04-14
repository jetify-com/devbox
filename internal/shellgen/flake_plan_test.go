package shellgen

import (
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
