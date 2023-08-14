package shellgen

import (
	"context"
	"runtime/trace"

	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/nix"
)

// flakePlan contains the data to populate the top level flake.nix file
// that builds the devbox environment
type flakePlan struct {
	BinaryCache string
	NixpkgsInfo *NixpkgsInfo
	FlakeInputs []*flakeInput
	Packages    []*devpkg.Package
	System      string
}

func newFlakePlan(ctx context.Context, devbox devboxer) (*flakePlan, error) {
	ctx, task := trace.NewTask(ctx, "devboxFlakePlan")
	defer task.End()

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

	packages, err := devbox.AllInstallablePackages()
	if err != nil {
		return nil, err
	}
	flakeInputs := flakeInputs(ctx, packages)
	nixpkgsInfo := getNixpkgsInfo(devbox.Config().NixPkgsCommitHash())

	// This is an optimization. Try to reuse the nixpkgs info from the flake
	// inputs to avoid introducing a new one.
	for _, input := range flakeInputs {
		if input.IsNixpkgs() {
			nixpkgsInfo = getNixpkgsInfo(input.HashFromNixPkgsURL())
			break
		}
	}

	return &flakePlan{
		BinaryCache: devpkg.BinaryCache,
		FlakeInputs: flakeInputs,
		NixpkgsInfo: nixpkgsInfo,
		Packages:    packages,
		System:      nix.System(),
	}, nil
}
