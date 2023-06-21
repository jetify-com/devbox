package filegen

import (
	"context"
	"runtime/trace"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/goutil"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

// FlakePlan contains the data to populate the top level flake.nix file
// that builds the devbox environment
type FlakePlan struct {
	NixpkgsInfo *plansdk.NixpkgsInfo
	FlakeInputs []*flakeInput
}

func newFlakePlan(ctx context.Context, devbox devboxer) (*FlakePlan, error) {
	ctx, task := trace.NewTask(ctx, "devboxFlakePlan")
	defer task.End()

	// Create plugin directories first because inputs might depend on them
	for _, pkg := range devbox.PackagesAsInputs() {
		if err := devbox.PluginManager().Create(pkg); err != nil {
			return nil, err
		}
	}

	for _, included := range devbox.Config().Include {
		// This is a slightly weird place to put this, but since includes can't be
		// added via command and we need them to be added before we call
		// plugin manager.Include
		if err := devbox.Lockfile().Add(included); err != nil {
			return nil, err
		}
		if err := devbox.PluginManager().Include(included); err != nil {
			return nil, err
		}
	}

	shellPlan := &FlakePlan{}
	var err error
	shellPlan.FlakeInputs, err = flakeInputs(ctx, devbox)
	if err != nil {
		return nil, err
	}

	nixpkgsInfo := plansdk.GetNixpkgsInfo(devbox.Config().NixPkgsCommitHash())

	// This is an optimization. Try to reuse the nixpkgs info from the flake
	// inputs to avoid introducing a new one.
	for _, input := range shellPlan.FlakeInputs {
		if input.IsNixpkgs() {
			nixpkgsInfo = plansdk.GetNixpkgsInfo(input.HashFromNixPkgsURL())
			break
		}
	}

	shellPlan.NixpkgsInfo = nixpkgsInfo

	return shellPlan, nil
}

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
	return plansdk.GetNixpkgsInfo(hash).URL
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

	inputs := map[string]*flakeInput{}

	userPackages := devbox.PackagesAsInputs()
	pluginPackages, err := devbox.PluginManager().PluginPackages(userPackages)
	if err != nil {
		return nil, err
	}

	order := []string{}
	// We prioritize plugin packages so that the php plugin works. Not sure
	// if this is behavior we want for user plugins. We may need to add an optional
	// priority field to the config.
	for _, pkg := range append(pluginPackages, userPackages...) {
		AttributePath, err := pkg.FullPackageAttributePath()
		if err != nil {
			return nil, err
		}
		if input, ok := inputs[pkg.URLForInput()]; !ok {
			order = append(order, pkg.URLForInput())
			inputs[pkg.URLForInput()] = &flakeInput{
				Name:     pkg.InputName(),
				URL:      pkg.URLForInput(),
				Packages: []string{AttributePath},
			}
		} else {
			input.Packages = lo.Uniq(
				append(inputs[pkg.URLForInput()].Packages, AttributePath),
			)
		}
	}

	return goutil.PickByKeysSorted(inputs, order), nil
}
