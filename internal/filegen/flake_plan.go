package filegen

import (
	"context"
	"runtime/trace"
)

// flakePlan contains the data to populate the top level flake.nix file
// that builds the devbox environment
type flakePlan struct {
	NixpkgsInfo *NixpkgsInfo
	FlakeInputs []*flakeInput
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

	shellPlan := &flakePlan{}
	var err error
	shellPlan.FlakeInputs, err = flakeInputs(ctx, devbox)
	if err != nil {
		return nil, err
	}

	nixpkgsInfo := getNixpkgsInfo(devbox.Config().NixPkgsCommitHash())

	// This is an optimization. Try to reuse the nixpkgs info from the flake
	// inputs to avoid introducing a new one.
	for _, input := range shellPlan.FlakeInputs {
		if input.IsNixpkgs() {
			nixpkgsInfo = getNixpkgsInfo(input.HashFromNixPkgsURL())
			break
		}
	}

	shellPlan.NixpkgsInfo = nixpkgsInfo

	return shellPlan, nil
}
