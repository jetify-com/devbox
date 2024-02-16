package shellgen

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime/trace"
	"strings"

	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/nix"
)

// flakePlan contains the data to populate the top level flake.nix file
// that builds the devbox environment
type flakePlan struct {
	BinaryCache string
	NixpkgsInfo *NixpkgsInfo
	Packages    []*devpkg.Package
	FlakeInputs []flakeInput
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

	// Fill the NarInfo Cache concurrently as a perf-optimization, prior to invoking
	// IsInBinaryCache in flakeInputs() and in the flake.nix template.
	if err := devpkg.FillNarInfoCache(ctx, packages...); err != nil {
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

func (f *flakePlan) needsGlibcPatch() bool {
	for _, in := range f.FlakeInputs {
		if in.URL == glibcPatchFlakeRef {
			return true
		}
	}
	return false
}

type glibcPatchFlake struct {
	// NixpkgsGlibcFlakeRef is a flake reference to the nixpkgs flake
	// containing the new glibc package.
	NixpkgsGlibcFlakeRef string

	// Inputs is the attribute set of flake inputs. The key is the input
	// name and the value is a flake reference.
	Inputs map[string]string

	// Outputs is the attribute set of flake outputs. It follows the
	// standard flake output schema of system.name = derivation. The
	// derivation can be any valid Nix expression.
	Outputs struct {
		Packages map[string]map[string]string
	}
}

func newGlibcPatchFlake(nixpkgsGlibcRev string, packages []*devpkg.Package) (glibcPatchFlake, error) {
	flake := glibcPatchFlake{NixpkgsGlibcFlakeRef: "flake:nixpkgs/" + nixpkgsGlibcRev}
	for _, pkg := range packages {
		if !pkg.PatchGlibc {
			continue
		}

		err := flake.addPackageOutput(pkg)
		if err != nil {
			return glibcPatchFlake{}, err
		}
	}
	return flake, nil
}

func (g *glibcPatchFlake) addPackageOutput(pkg *devpkg.Package) error {
	if g.Inputs == nil {
		g.Inputs = make(map[string]string)
	}
	inputName := pkg.FlakeInputName()
	g.Inputs[inputName] = pkg.URLForFlakeInput()

	attrPath, err := pkg.FullPackageAttributePath()
	if err != nil {
		return err
	}
	// Remove the legacyPackages.<system> prefix.
	outputName := strings.SplitN(attrPath, ".", 3)[2]

	if g.Outputs.Packages == nil {
		g.Outputs.Packages = map[string]map[string]string{nix.System(): {}}
	}
	if cached, err := pkg.IsInBinaryCache(); err == nil && cached {
		if expr, err := g.fetchClosureExpr(pkg); err == nil {
			g.Outputs.Packages[nix.System()][outputName] = expr
			return nil
		}
	}
	g.Outputs.Packages[nix.System()][outputName] = strings.Join([]string{"pkgs", inputName, nix.System(), outputName}, ".")
	return nil
}

func (g *glibcPatchFlake) fetchClosureExpr(pkg *devpkg.Package) (string, error) {
	storePath, err := pkg.InputAddressedPath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`builtins.fetchClosure {
  fromStore = "%s";
  fromPath = "%s";
  inputAddressed = true;
}`, devpkg.BinaryCache, storePath), nil
}

func (g *glibcPatchFlake) writeTo(dir string) error {
	err := writeFromTemplate(dir, g, "glibc-patch.nix", "flake.nix")
	if err != nil {
		return err
	}
	return writeGlibcPatchScript(filepath.Join(dir, "glibc-patch.bash"))
}
