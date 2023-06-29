package shellgen

import (
	"context"
	"runtime/trace"

	"go.jetpack.io/devbox/internal/nix"
)

// flakePlan contains the data to populate the top level flake.nix file
// that builds the devbox environment
type flakePlan struct {
	BinaryCacheStore string
	NixpkgsInfo      *NixpkgsInfo
	FlakeInputs      []*flakeInput
	Packages         []*nix.Package
	System           string
}

func newFlakePlan(ctx context.Context, devbox devboxer) (*flakePlan, error) {
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

	userPackages := devbox.PackagesAsInputs()
	pluginPackages, err := devbox.PluginManager().PluginInputs(userPackages)
	if err != nil {
		return nil, err
	}
	// We prioritize plugin packages so that the php plugin works. Not sure
	// if this is behavior we want for user plugins. We may need to add an optional
	// priority field to the config.
	packages := append(pluginPackages, userPackages...)

	flakeInputs, err := flakeInputs(ctx, packages)
	if err != nil {
		return nil, err
	}

	nixpkgsInfo := getNixpkgsInfo(devbox.Config().NixPkgsCommitHash())

	// This is an optimization. Try to reuse the nixpkgs info from the flake
	// inputs to avoid introducing a new one.
	for _, input := range flakeInputs {
		if input.IsNixpkgs() {
			nixpkgsInfo = getNixpkgsInfo(input.HashFromNixPkgsURL())
			break
		}
	}

	system, err := nix.System()
	if err != nil {
		return nil, err
	}

	return &flakePlan{
		BinaryCacheStore: nix.BinaryCacheStore,
		FlakeInputs:      flakeInputs,
		NixpkgsInfo:      nixpkgsInfo,
		Packages:         packages,
		System:           system,
	}, nil
}
