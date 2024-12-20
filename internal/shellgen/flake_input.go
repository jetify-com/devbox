package shellgen

import (
	"context"
	"errors"
	"log/slog"
	"runtime/trace"
	"slices"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/nix/flake"
)

var glibcPatchFlakeRef = flake.Ref{Type: flake.TypePath, Path: "./glibc-patch"}

type flakeInput struct {
	Name     string
	Packages []*devpkg.Package
	Ref      flake.Ref
}

// IsNixpkgs reports whether the input looks like a nixpkgs flake.
//
// While there are many ways to specify this input, devbox always uses
// github:NixOS/nixpkgs/<hash> as the URL. If the user wishes to reference nixpkgs
// themselves, this function may not return true.
func (f *flakeInput) IsNixpkgs() bool {
	switch f.Ref.Type {
	case flake.TypeGitHub:
		return f.Ref.Owner == "NixOS" && f.Ref.Repo == "nixpkgs"
	case flake.TypeIndirect:
		return f.Ref.ID == "nixpkgs"
	default:
		return false
	}
}

func (f *flakeInput) HashFromNixPkgsURL() string {
	return f.Ref.Rev
}

func (f *flakeInput) URLWithCaching() string {
	if !f.IsNixpkgs() {
		return f.Ref.String()
	}
	return getNixpkgsInfo(f.Ref.Rev).URL
}

func (f *flakeInput) PkgImportName() string {
	return f.Name + "-pkgs"
}

type SymlinkJoin struct {
	Name  string
	Paths []string
}

// BuildInputsForSymlinkJoin returns a list of SymlinkJoin objects that can be used
// as the buildInput. Used for packages that have non-default outputs that need to
// be combined into a single buildInput.
func (f *flakeInput) BuildInputsForSymlinkJoin() ([]*SymlinkJoin, error) {
	joins := []*SymlinkJoin{}
	for _, pkg := range f.Packages {

		// Skip packages that don't need a symlink join.
		if needs, err := needsSymlinkJoin(pkg); err != nil {
			return nil, err
		} else if !needs {
			continue
		}

		// Skip packages that are already in the binary cache. These will be directly
		// included in the buildInputs using `builtins.fetchClosure` of their store paths.
		inCache, err := pkg.IsInBinaryCache()
		if err != nil {
			return nil, err
		}
		if inCache {
			continue
		}

		attributePath, err := pkg.FullPackageAttributePath()
		if err != nil {
			return nil, err
		}

		if pkg.Patch {
			return nil, errors.New("patch_glibc is not yet supported for packages with non-default outputs")
		}

		outputNames, err := pkg.GetOutputNames()
		if err != nil {
			return nil, err
		}

		joins = append(joins, &SymlinkJoin{
			Name: pkg.String() + "-combined",
			Paths: lo.Map(outputNames, func(outputName string, _ int) string {
				if !f.IsNixpkgs() {
					return f.Name + "." + attributePath + "." + outputName
				}
				parts := strings.Split(attributePath, ".")
				return f.PkgImportName() + "." + strings.Join(parts[2:], ".") + "." + outputName
			}),
		})
	}
	return joins, nil
}

func (f *flakeInput) BuildInputs() ([]string, error) {
	var err error

	// Skip packages that will be handled in BuildInputsForSymlinkJoin
	packages := []*devpkg.Package{}
	for _, pkg := range f.Packages {
		if needs, err := needsSymlinkJoin(pkg); err != nil {
			return nil, err
		} else if !needs {
			packages = append(packages, pkg)
		}
	}

	attributePaths := lo.Map(packages, func(pkg *devpkg.Package, _ int) string {
		attributePath, attributePathErr := pkg.FullPackageAttributePath()
		if attributePathErr != nil {
			err = attributePathErr
		}
		if pkg.Patch {
			// When the package comes from the glibc flake, the
			// "legacyPackages" portion of the attribute path
			// becomes just "packages" (matching the standard flake
			// output schema).
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

	var flakeInputs keyedSlice
	for _, pkg := range packages {
		// Non-nix packages (e.g. runx) don't belong in the flake
		if !pkg.IsNix() {
			continue
		}

		// Don't include cached packages (like local or remote flakes)
		// that can be fetched from a Binary Cache Store.
		// TODO(savil): return error?
		cached, err := pkg.IsInBinaryCache()
		if err != nil {
			slog.Error("error checking if package is in binary cache", "err", err)
		}
		if err == nil && cached {
			continue
		}

		// Packages that need a glibc patch are assigned to the special
		// glibc-patched flake input. This input refers to the
		// glibc-patch.nix flake.
		if pkg.Patch {
			nixpkgsGlibc := flakeInputs.getOrAppend(glibcPatchFlakeRef.String())
			nixpkgsGlibc.Name = "glibc-patch"
			nixpkgsGlibc.Ref = glibcPatchFlakeRef
			nixpkgsGlibc.Packages = append(nixpkgsGlibc.Packages, pkg)
			continue
		}

		installable, err := pkg.FlakeInstallable()
		if err != nil {
			// I don't think this should happen at this point. The
			// packages should already be resolved to valid nixpkgs
			// packages.
			slog.Debug("error resolving package to flake installable", "err", err)
			continue
		}
		flake := flakeInputs.getOrAppend(installable.Ref.String())
		flake.Name = pkg.FlakeInputName()
		flake.Ref = installable.Ref

		// TODO(gcurtis): is the uniqueness check necessary? We're
		// comparing pointers.
		if !slices.Contains(flake.Packages, pkg) {
			flake.Packages = append(flake.Packages, pkg)
		}
	}
	return flakeInputs.slice
}

// keyedSlice keys the elements of an append-only slice for fast lookups.
type keyedSlice struct {
	slice  []flakeInput
	lookup map[string]int
}

// getOrAppend returns a pointer to the slice element with a given key. If the
// key doesn't exist, a new element is automatically appended to the slice. The
// pointer is valid until the next append.
func (k *keyedSlice) getOrAppend(key string) *flakeInput {
	if k.lookup == nil {
		k.lookup = make(map[string]int)
	}
	if i, ok := k.lookup[key]; ok {
		return &k.slice[i]
	}
	k.slice = append(k.slice, flakeInput{})
	k.lookup[key] = len(k.slice) - 1
	return &k.slice[len(k.slice)-1]
}

// needsSymlinkJoin is used to filter packages with multiple outputs.
// Multiple outputs -> SymlinkJoin.
// Single or no output -> directly use in buildInputs
func needsSymlinkJoin(pkg *devpkg.Package) (bool, error) {
	outputNames, err := pkg.GetOutputNames()
	if err != nil {
		return false, err
	}
	return len(outputNames) > 1, nil
}
