package shellgen

import (
	"context"
	"runtime/trace"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/goutil"
	"go.jetpack.io/devbox/internal/nix"
)

type flakeInput struct {
	Name     string
	Packages []string
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

func (f *flakeInput) BuildInputs() []string {
	if !f.IsNixpkgs() {
		return lo.Map(f.Packages, func(pkg string, _ int) string {
			return f.Name + "." + pkg
		})
	}
	return lo.Map(f.Packages, func(pkg string, _ int) string {
		parts := strings.Split(pkg, ".")
		// Ugh, not sure if this is reliable?
		return f.PkgImportName() + "." + strings.Join(parts[2:], ".")
	})
}

// flakeInputs returns a list of flake inputs for the top level flake.nix
// created by devbox. We map packages to the correct flake and attribute path
// and group flakes by URL to avoid duplication. All inputs should be locked
// i.e. have a commit hash and always resolve to the same package/version.
// Note: inputs returned by this function include plugin packages. (php only for now)
// It's not entirely clear we always want to add plugin packages to the top level
func flakeInputs(ctx context.Context, devbox devboxer) ([]*flakeInput, error) {
	defer trace.StartRegion(ctx, "flakeInputs").End()

	// Use the verbose name flakeInputs to distinguish from `inputs`
	// which refer to `nix.Input` in most of the codebase.
	flakeInputs := map[string]*flakeInput{}

	userInputs := devbox.PackagesAsInputs()
	pluginInputs, err := devbox.PluginManager().PluginInputs(userInputs)
	if err != nil {
		return nil, err
	}

	order := []string{}
	// We prioritize plugin packages so that the php plugin works. Not sure
	// if this is behavior we want for user plugins. We may need to add an optional
	// priority field to the config.
	for _, input := range append(pluginInputs, userInputs...) {
		AttributePath, err := input.FullPackageAttributePath()
		if err != nil {
			return nil, err
		}
		if flkInput, ok := flakeInputs[input.URLForFlakeInput()]; !ok {
			order = append(order, input.URLForFlakeInput())
			flakeInputs[input.URLForFlakeInput()] = &flakeInput{
				Name:     input.FlakeInputName(),
				URL:      input.URLForFlakeInput(),
				Packages: []string{AttributePath},
			}
		} else {
			flkInput.Packages = lo.Uniq(
				append(flakeInputs[input.URLForFlakeInput()].Packages, AttributePath),
			)
		}
	}

	return goutil.PickByKeysSorted(flakeInputs, order), nil
}
