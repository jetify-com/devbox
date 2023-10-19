package shellgen

import (
	"context"
	"runtime/trace"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/goutil"
	"go.jetpack.io/devbox/internal/nix"
)

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

type buildInput struct {
	AttrPath   string
	PatchGlibc bool
}

func (f *flakeInput) BuildInputs() ([]buildInput, error) {
	inputs := make([]buildInput, len(f.Packages))
	prefix := f.Name
	if f.IsNixpkgs() {
		prefix = f.PkgImportName()
	}
	prefix += "."
	for i, pkg := range f.Packages {
		attrPath, err := pkg.FullPackageAttributePath()
		if err != nil {
			return nil, err
		}
		if f.IsNixpkgs() {
			// Remove the legacyPackages.<system> prefix.
			attrPath = strings.SplitN(attrPath, ".", 3)[2]
		}
		inputs[i] = buildInput{
			AttrPath:   prefix + attrPath,
			PatchGlibc: pkg.PatchGlibc,
		}
	}
	return inputs, nil
}

// flakeInputs returns a list of flake inputs for the top level flake.nix
// created by devbox. We map packages to the correct flake and attribute path
// and group flakes by URL to avoid duplication. All inputs should be locked
// i.e. have a commit hash and always resolve to the same package/version.
// Note: inputs returned by this function include plugin packages. (php only for now)
// It's not entirely clear we always want to add plugin packages to the top level
func flakeInputs(ctx context.Context, packages []*devpkg.Package) []*flakeInput {
	defer trace.StartRegion(ctx, "flakeInputs").End()

	// Use the verbose name flakeInputs to distinguish from `inputs`
	// which refer to `nix.Input` in most of the codebase.
	flakeInputs := map[string]*flakeInput{}

	packages = lo.Filter(packages, func(item *devpkg.Package, _ int) bool {
		// Non nix packages (e.g. runx) don't belong in the flake
		if !item.IsNix() {
			return false
		}

		// Include packages (like local or remote flakes) that cannot be
		// fetched from a Binary Cache Store.
		if !featureflag.RemoveNixpkgs.Enabled() {
			return true
		}

		inCache, err := item.IsInBinaryCache()
		if err != nil {
			// Ignore this error for now. TODO savil: return error?
			return true
		}
		return !inCache
	})

	order := []string{}
	for _, pkg := range packages {
		if flkInput, ok := flakeInputs[pkg.URLForFlakeInput()]; !ok {
			order = append(order, pkg.URLForFlakeInput())
			flakeInputs[pkg.URLForFlakeInput()] = &flakeInput{
				Name:     pkg.FlakeInputName(),
				URL:      pkg.URLForFlakeInput(),
				Packages: []*devpkg.Package{pkg},
			}
		} else {
			flkInput.Packages = lo.Uniq(
				append(flakeInputs[pkg.URLForFlakeInput()].Packages, pkg),
			)
		}
	}

	return goutil.PickByKeysSorted(flakeInputs, order)
}
