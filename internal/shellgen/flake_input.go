package shellgen

import (
	"context"
	"runtime/trace"
	"slices"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/nix"
)

const glibcPatchFlakeRef = "path:./glibc-patch"

type flakeInput struct {
	Name     string
	Packages []*devpkg.Package
	URL      string
}

// IsNixpkgs returns true if the input is a nixpkgs flake of the form:
// github:NixOS/nixpkgs/...
//
// While there are many ways to specify this input, devbox always uses
// github:NixOS/nixpkgs/<hash> as the URL. If the user wishes to reference nixpkgs
// themselves, this function may not return true.
func (f *flakeInput) IsNixpkgs() bool {
	return nix.IsGithubNixpkgsURL(f.URL)
}

func (f *flakeInput) HashFromNixPkgsURL() string {
	if !f.IsNixpkgs() {
		return ""
	}
	return nix.HashFromNixPkgsURL(f.URL)
}

func (f *flakeInput) URLWithCaching() string {
	if !f.IsNixpkgs() {
		return f.URL
	}
	hash := nix.HashFromNixPkgsURL(f.URL)
	return getNixpkgsInfo(hash).URL
}

func (f *flakeInput) PkgImportName() string {
	return f.Name + "-pkgs"
}

func (f *flakeInput) BuildInputs() ([]string, error) {
	var err error
	attributePaths := lo.Map(f.Packages, func(pkg *devpkg.Package, _ int) string {
		attributePath, attributePathErr := pkg.FullPackageAttributePath()
		if attributePathErr != nil {
			err = attributePathErr
		}
		if pkg.PatchGlibc {
			return strings.Replace(attributePath, "legacyPackages", "packages", 1)
		}
		return attributePath
	})
	if err != nil {
		return nil, err
	}
	if !f.IsNixpkgs() {
		return lo.Map(attributePaths, func(pkg string, _ int) string {
			return f.Name + "." + pkg
		}), nil
	}
	return lo.Map(attributePaths, func(pkg string, _ int) string {
		parts := strings.Split(pkg, ".")
		// Ugh, not sure if this is reliable?
		return f.PkgImportName() + "." + strings.Join(parts[2:], ".")
	}), nil
}

// flakeInputs returns a list of flake inputs for the top level flake.nix
// created by devbox. We map packages to the correct flake and attribute path
// and group flakes by URL to avoid duplication. All inputs should be locked
// i.e. have a commit hash and always resolve to the same package/version.
// Note: inputs returned by this function include plugin packages. (php only for now)
// It's not entirely clear we always want to add plugin packages to the top level
func flakeInputs(ctx context.Context, packages []*devpkg.Package) []flakeInput {
	defer trace.StartRegion(ctx, "flakeInputs").End()

	var flakeInputs []flakeInput
	flakeInputsByURL := make(map[string]*flakeInput)
	for _, pkg := range packages {
		// Non-nix packages (e.g. runx) don't belong in the flake
		if !pkg.IsNix() {
			continue
		}

		// Don't include cached packages (like local or remote flakes)
		// that can be fetched from a Binary Cache Store.
		if featureflag.RemoveNixpkgs.Enabled() {
			// TODO(savil): return error?
			cached, err := pkg.IsInBinaryCache()
			if err != nil {
				debug.Log("error checking if package is in binary cache: %v", err)
			}
			if err == nil && cached {
				continue
			}
		}

		// Packages that need a glibc patch are assigned to the special
		// glibc-patched flake input. This input refers to the
		// glibc-patch.nix flake.
		if pkg.PatchGlibc {
			nixpkgsGlibc := flakeInputsByURL[glibcPatchFlakeRef]
			if nixpkgsGlibc == nil {
				flakeInputs = append(flakeInputs, flakeInput{
					Name:     "glibc-patch",
					URL:      glibcPatchFlakeRef,
					Packages: []*devpkg.Package{pkg},
				})
				flakeInputsByURL[glibcPatchFlakeRef] = &flakeInputs[len(flakeInputs)-1]
				continue
			}
			nixpkgsGlibc.Packages = append(nixpkgsGlibc.Packages, pkg)
			continue
		}

		pkgURL := pkg.URLForFlakeInput()
		existing := flakeInputsByURL[pkgURL]
		if existing == nil {
			flakeInputs = append(flakeInputs, flakeInput{
				Name:     pkg.FlakeInputName(),
				URL:      pkgURL,
				Packages: []*devpkg.Package{pkg},
			})
			flakeInputsByURL[pkgURL] = &flakeInputs[len(flakeInputs)-1]
			continue
		}

		// TODO(gcurtis): is the uniqueness check necessary? We're
		// comparing pointers.
		if !slices.Contains(existing.Packages, pkg) {
			existing.Packages = append(existing.Packages, pkg)
		}
	}
	return flakeInputs
}
